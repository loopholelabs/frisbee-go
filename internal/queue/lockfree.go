/*
	Copyright 2022 Loophole Labs
	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at
		   http://www.apache.org/licenses/LICENSE-2.0
	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package queue

import (
	"runtime"
	"sync/atomic"
)

type Pointer[T any] interface {
	*T
}

// node is a struct that keeps track of its own position as well as a piece of data
// stored as an unsafe.Pointer. Normally we would store the pointer to a packet.Packet
// directly, however benchmarking shows performance improvements with unsafe.Pointer instead
type node[T any, P Pointer[T]] struct {
	_padding0 [8]uint64 //nolint:structcheck,unused
	position  uint64
	_padding1 [8]uint64 //nolint:structcheck,unused
	data      P
}

// nodes is a struct type containing a slice of node pointers
type nodes[T any, P Pointer[T]] []*node[T, P]

// LockFree is the struct used to store a blocking or non-blocking FIFO queue of type *packet.Packet
//
// In it's non-blocking form it acts as a ringbuffer, overwriting old data when new data arrives. In its blocking
// form it waits for a space in the queue to open up before it adds the item to the LockFree.
type LockFree[T any, P Pointer[T]] struct {
	_padding0 [8]uint64 //nolint:structcheck,unused
	head      uint64
	_padding1 [8]uint64 //nolint:structcheck,unused
	tail      uint64
	_padding2 [8]uint64 //nolint:structcheck,unused
	mask      uint64
	_padding3 [8]uint64 //nolint:structcheck,unused
	closed    uint64
	_padding4 [8]uint64 //nolint:structcheck,unused
	nodes     []*node[T, P]
	_padding5 [8]uint64 //nolint:structcheck,unused
	overflow  func() (uint64, error)
}

// NewLockFree creates a new LockFree with blocking or non-blocking behavior
func NewLockFree[T any, P Pointer[T]](size uint64) *LockFree[T, P] {
	q := new(LockFree[T, P])
	if size < 1 {
		size = 1
	}
	q.overflow = q.blocker
	q.init(size)
	return q
}

// init actually initializes a queue and can be used in the future to reuse LockFree structs
// with their own pool
func (q *LockFree[T, P]) init(size uint64) {
	size = round(size)
	q.nodes = make(nodes[T, P], size)
	for i := uint64(0); i < size; i++ {
		q.nodes[i] = &node[T, P]{position: i}
	}
	q.mask = size - 1
}

// blocker is a LockFree.overflow function that blocks a Push operation from
// proceeding if the LockFree is ever full of data.
//
// If two Push operations happen simultaneously, blocker will block both of them until
// a Pop takes place, and unblock both of them at the same time. This can cause problems,
// however in our use case it won't because there shouldn't ever be more than one producer
// operating on the LockFree at any given time. There may be multiple consumers in the future,
// but that won't cause any problems.
//
// If we decide to use this as an MPMC LockFree instead of a SPMC LockFree (which is how we currently use it)
// then we can solve this bug by replacing the existing `default` switch case in the Push function with the
// following snippet:
// ```
// default:
//			head, err = q.overflow()
//			if err != nil {
//				return err
//			}
// ```
func (q *LockFree[T, P]) blocker() (head uint64, err error) {
LOOP:
	head = atomic.LoadUint64(&q.head)
	if uint64(len(q.nodes)) == head-atomic.LoadUint64(&q.tail) {
		if atomic.LoadUint64(&q.closed) == 1 {
			err = Closed
			return
		}
		runtime.Gosched()
		goto LOOP
	}
	return
}

// Push appends an item of type *packet.Packet to the LockFree, and will block
// until the item is pushed succ›essfully (with the blocking function depending
// on whether this is a blocking LockFree).
//
// This method is not meant to be used concurrently, and the LockFree is meant to operate
// as an SPMC LockFree with one producer operating at a time. If we want to use this as an MPMC LockFree
// we can modify this Push function by replacing the existing `default` switch case with the
// following snippet:
// ```
// default:
//			head, err = q.overflow()
//			if err != nil {
//				return err
//			}
// ```
func (q *LockFree[T, P]) Push(item P) error {
	var newNode *node[T, P]
	head, err := q.overflow()
	if err != nil {
		return err
	}
RETRY:
	for {
		if atomic.LoadUint64(&q.closed) == 1 {
			return Closed
		}

		newNode = q.nodes[head&q.mask]
		switch dif := atomic.LoadUint64(&newNode.position) - head; {
		case dif == 0:
			if atomic.CompareAndSwapUint64(&q.head, head, head+1) {
				break RETRY
			}
		default:
			head = atomic.LoadUint64(&q.head)
		}
		runtime.Gosched()
	}
	newNode.data = item
	atomic.StoreUint64(&newNode.position, head+1)
	return nil
}

// Pop removes an item from the start of the LockFree and returns it to the caller.
// This method blocks until an item is available, but unblocks when the LockFree is closed.
// This allows for long-term listeners to wait on the LockFree until either an item is available
// or the LockFree is closed.
//
// This method is safe to be used concurrently and is even optimized for the SPMC use case.
func (q *LockFree[T, P]) Pop() (P, error) {
	var oldNode *node[T, P]
	var oldPosition = atomic.LoadUint64(&q.tail)
RETRY:
	if atomic.LoadUint64(&q.closed) == 1 {
		return nil, Closed
	}

	oldNode = q.nodes[oldPosition&q.mask]
	switch dif := atomic.LoadUint64(&oldNode.position) - (oldPosition + 1); {
	case dif == 0:
		if atomic.CompareAndSwapUint64(&q.tail, oldPosition, oldPosition+1) {
			goto DONE
		}
	default:
		oldPosition = atomic.LoadUint64(&q.tail)
	}
	runtime.Gosched()
	goto RETRY
DONE:
	data := oldNode.data
	oldNode.data = nil
	atomic.StoreUint64(&oldNode.position, oldPosition+q.mask+1)
	return data, nil
}

// Close marks the LockFree as closed, returns any waiting Pop() calls,
// and blocks all future Push calls from occurring.
func (q *LockFree[T, P]) Close() {
	atomic.CompareAndSwapUint64(&q.closed, 0, 1)
}

// IsClosed returns whether the LockFree has been closed
func (q *LockFree[T, P]) IsClosed() bool {
	return atomic.LoadUint64(&q.closed) == 1
}

// Length is the current number of items in the LockFree
func (q *LockFree[T, P]) Length() int {
	return int(atomic.LoadUint64(&q.head) - atomic.LoadUint64(&q.tail))
}

// Drain drains all the current packets in the queue and returns them to the caller.
//
// It is an unsafe function that should only be used once, only after the queue has been closed,
// and only while there are no producers writing to it. If used incorrectly it has the potential
// to infinitely block the caller. If used correctly, it allows a single caller to drain any remaining
// packets in the queue after the queue has been closed.
func (q *LockFree[T, P]) Drain() []P {
	length := q.Length()
	packets := make([]P, 0, length)
	for i := 0; i < length; i++ {
		var oldNode *node[T, P]
		var oldPosition = atomic.LoadUint64(&q.tail)
	RETRY:
		oldNode = q.nodes[oldPosition&q.mask]
		switch dif := atomic.LoadUint64(&oldNode.position) - (oldPosition + 1); {
		case dif == 0:
			if atomic.CompareAndSwapUint64(&q.tail, oldPosition, oldPosition+1) {
				goto DONE
			}
		default:
			oldPosition = atomic.LoadUint64(&q.tail)
		}
		runtime.Gosched()
		goto RETRY
	DONE:
		data := oldNode.data
		oldNode.data = nil
		atomic.StoreUint64(&oldNode.position, oldPosition+q.mask+1)
		packets = append(packets, data)
	}
	return packets
}

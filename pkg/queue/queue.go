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
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/pkg/errors"
	"runtime"
	"sync/atomic"
	"unsafe"
)

var (
	Closed = errors.New("queue is closed")
)

func round(value uint64) uint64 {
	value--
	value |= value >> 1
	value |= value >> 2
	value |= value >> 4
	value |= value >> 8
	value |= value >> 16
	value |= value >> 32
	value++
	return value
}

type node struct {
	position uint64
	data     unsafe.Pointer
}

type nodes []*node

type Queue struct {
	head      uint64
	_padding0 [8]uint64 //nolint:structcheck,unused
	tail      uint64
	_padding1 [8]uint64 //nolint:structcheck,unused
	mask      uint64
	_padding2 [8]uint64 //nolint:structcheck,unused
	closed    uint64
	_padding3 [8]uint64 //nolint:structcheck,unused
	nodes     nodes
	_padding4 [8]uint64 //nolint:structcheck,unused
	overflow  func(uint64) error
}

func New(size uint64, blocking bool) *Queue {
	q := new(Queue)
	if size < 1 {
		size = 1
	}
	if blocking {
		q.overflow = q.blocker
	} else {
		q.overflow = q.unblocker
	}
	q.init(size)
	return q
}

func (q *Queue) init(size uint64) {
	size = round(size)
	q.nodes = make(nodes, size)
	for i := uint64(0); i < size; i++ {
		q.nodes[i] = &node{position: i}
	}
	q.mask = size - 1
}

func (q *Queue) blocker(head uint64) error {
LOOP:
	tail := atomic.LoadUint64(&q.tail)
	if uint64(len(q.nodes)) == head-tail {
		if atomic.LoadUint64(&q.closed) == 1 {
			return Closed
		}
		runtime.Gosched()
		goto LOOP
	}
	return nil
}

func (q *Queue) unblocker(head uint64) error {
	if uint64(len(q.nodes)) == head-atomic.LoadUint64(&q.tail) {
		p, err := q.Pop()
		packet.Put((*packet.Packet)(p))
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *Queue) Push(item unsafe.Pointer) error {
	var newNode *node
	head := atomic.LoadUint64(&q.head)
	err := q.blocker(head)
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

func (q *Queue) Pop() (unsafe.Pointer, error) {
	var oldNode *node
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

func (q *Queue) Close() {
	atomic.CompareAndSwapUint64(&q.closed, 0, 1)
}

func (q *Queue) IsClosed() bool {
	return atomic.LoadUint64(&q.closed) == 1
}

func (q *Queue) Length() int {
	return int(atomic.LoadUint64(&q.head) - atomic.LoadUint64(&q.tail))
}

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
	"sync"
)

// Circular is a circular sized FIFO queue that uses
// an array of fixed size to store the elements.
//
// It is thread safe and extremely performant, however
// it is a blocking queue and will block the caller
// if the queue is full or if it is empty.
type Circular[T any, P Pointer[T]] struct {
	_padding0 [8]uint64 //nolint:structcheck,unused
	head      uint64
	_padding1 [8]uint64 //nolint:structcheck,unused
	tail      uint64
	_padding2 [8]uint64 //nolint:structcheck,unused
	maxSize   uint64
	_padding3 [8]uint64 //nolint:structcheck,unused
	closed    bool
	_padding4 [8]uint64 //nolint:structcheck,unused
	lock      *sync.Mutex
	_padding5 [8]uint64 //nolint:structcheck,unused
	notEmpty  *sync.Cond
	_padding6 [8]uint64 //nolint:structcheck,unused
	notFull   *sync.Cond
	_padding7 [8]uint64 //nolint:structcheck,unused
	nodes     []P
}

// NewCircular creates a new circular queue with the given size.
func NewCircular[T any, P Pointer[T]](maxSize uint64) *Circular[T, P] {
	q := new(Circular[T, P])
	q.lock = new(sync.Mutex)
	q.notFull = sync.NewCond(q.lock)
	q.notEmpty = sync.NewCond(q.lock)

	q.head = 0
	q.tail = 0
	maxSize++
	if maxSize < 2 {
		q.maxSize = 2
	} else {
		q.maxSize = round(maxSize)
	}

	q.nodes = make([]P, q.maxSize)
	return q
}

// IsEmpty returns true if the queue is empty.
func (q *Circular[T, P]) IsEmpty() (empty bool) {
	q.lock.Lock()
	empty = q.isEmpty()
	q.lock.Unlock()
	return
}

// isEmpty is an internal function used to check if the
// queue is empty.
func (q *Circular[T, P]) isEmpty() bool {
	return q.head == q.tail
}

// IsFull returns true if the queue is full.
func (q *Circular[T, P]) IsFull() (full bool) {
	q.lock.Lock()
	full = q.isFull()
	q.lock.Unlock()
	return
}

// isFull is an internal function used to check if the
// queue is full.
func (q *Circular[T, P]) isFull() bool {
	return q.head == (q.tail+1)%q.maxSize
}

// IsClosed returns true if the queue is Closed
//
// The Drain method can be used to drain the queue after it is closed.
func (q *Circular[T, P]) IsClosed() (closed bool) {
	q.lock.Lock()
	closed = q.isClosed()
	q.lock.Unlock()
	return
}

// isClosed is an internal function used to check if the
// queue is closed.
func (q *Circular[T, P]) isClosed() bool {
	return q.closed
}

// Length returns the number of elements in the queue.
func (q *Circular[T, P]) Length() (size int) {
	q.lock.Lock()
	size = q.length()
	q.lock.Unlock()
	return
}

// length is an internal function used to get the number of elements in the queue.
func (q *Circular[T, P]) length() int {
	if q.tail < q.head {
		return int(q.maxSize - q.head + q.tail)
	}
	return int(q.tail - q.head)
}

// Close closes the queue permanently.
//
// The Drain method can be used to drain the queue after it is closed.
func (q *Circular[T, P]) Close() {
	q.lock.Lock()
	q.closed = true
	q.notFull.Broadcast()
	q.notEmpty.Broadcast()
	q.lock.Unlock()
}

// Push adds an element to the queue.
func (q *Circular[T, P]) Push(p P) error {
	q.lock.Lock()
LOOP:
	if q.isClosed() {
		q.lock.Unlock()
		return Closed
	}
	if q.isFull() {
		q.notFull.Wait()
		goto LOOP
	}

	q.nodes[q.tail] = p
	q.tail = (q.tail + 1) % q.maxSize
	q.notEmpty.Signal()
	q.lock.Unlock()
	return nil
}

// Pop removes an element from the queue.
func (q *Circular[T, P]) Pop() (p P, err error) {
	q.lock.Lock()
LOOP:
	if q.isClosed() {
		q.lock.Unlock()
		return nil, Closed
	}
	if q.isEmpty() {
		q.notEmpty.Wait()
		goto LOOP
	}

	p = q.nodes[q.head]
	q.head = (q.head + 1) % q.maxSize
	q.notFull.Signal()
	q.lock.Unlock()
	return
}

// Drain removes all elements from the queue.
// and returns them in a slice.
//
// This function should only be called after the queue is closed.
func (q *Circular[T, P]) Drain() (values []P) {
	q.lock.Lock()
	if q.isEmpty() {
		q.lock.Unlock()
		return nil
	}
	if size := int(q.head) - int(q.tail); size > 0 {
		values = make([]P, 0, size)
	} else {
		values = make([]P, 0, -1*size)
	}
	for i := 0; i < cap(values); i++ {
		values = append(values, q.nodes[q.head])
		q.head = (q.head + 1) % q.maxSize
	}
	q.lock.Unlock()
	return values
}

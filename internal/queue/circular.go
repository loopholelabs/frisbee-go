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
	"sync"
	"unsafe"
)

type Circular struct {
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
	nodes     []unsafe.Pointer
}

func NewCircular(maxSize uint64) *Circular {
	q := &Circular{}
	q.lock = &sync.Mutex{}
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

	q.nodes = make([]unsafe.Pointer, q.maxSize)
	return q
}

func (q *Circular) IsEmpty() (empty bool) {
	q.lock.Lock()
	empty = q.isEmpty()
	q.lock.Unlock()
	return
}

func (q *Circular) isEmpty() bool {
	return q.head == q.tail
}

func (q *Circular) IsFull() (full bool) {
	q.lock.Lock()
	full = q.isFull()
	q.lock.Unlock()
	return
}

func (q *Circular) isFull() bool {
	return q.head == (q.tail+1)%q.maxSize
}

func (q *Circular) IsClosed() (closed bool) {
	q.lock.Lock()
	closed = q.isClosed()
	q.lock.Unlock()
	return
}

func (q *Circular) isClosed() bool {
	return q.closed
}

func (q *Circular) Length() (size int) {
	q.lock.Lock()
	size = q.length()
	q.lock.Unlock()
	return
}

func (q *Circular) length() int {
	if q.tail < q.head {
		return int(q.maxSize - q.head + q.tail)
	}
	return int(q.tail - q.head)
}

func (q *Circular) Close() {
	q.lock.Lock()
	q.closed = true
	q.notFull.Broadcast()
	q.notEmpty.Broadcast()
	q.lock.Unlock()
}

func (q *Circular) Push(p *packet.Packet) error {
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

	q.nodes[q.tail] = unsafe.Pointer(p)
	q.tail = (q.tail + 1) % q.maxSize
	q.notEmpty.Signal()
	q.lock.Unlock()
	return nil
}

func (q *Circular) Pop() (p *packet.Packet, err error) {
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

	p = (*packet.Packet)(q.nodes[q.head])
	q.head = (q.head + 1) % q.maxSize
	q.notFull.Signal()
	q.lock.Unlock()
	return
}

func (q *Circular) Drain() (packets []*packet.Packet) {
	q.lock.Lock()
	if q.isEmpty() {
		q.lock.Unlock()
		return nil
	}
	if size := int(q.head) - int(q.tail); size > 0 {
		packets = make([]*packet.Packet, 0, size)
	} else {
		packets = make([]*packet.Packet, 0, -1*size)
	}
	for i := 0; i < cap(packets); i++ {
		packets = append(packets, (*packet.Packet)(q.nodes[q.head]))
		q.head = (q.head + 1) % q.maxSize
	}
	q.lock.Unlock()
	return packets
}

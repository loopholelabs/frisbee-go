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

package lqueue

import (
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/pkg/errors"
	"sync"
	"unsafe"
)

var (
	Closed = errors.New("queue is closed")
)

type LQueue struct {
	head    uint32
	tail    uint32
	maxSize uint32

	closed   bool
	lock     *sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond

	nodes []unsafe.Pointer
}

func New(maxSize uint32) *LQueue {
	queue := &LQueue{}
	queue.lock = &sync.Mutex{}
	queue.notFull = sync.NewCond(queue.lock)
	queue.notEmpty = sync.NewCond(queue.lock)

	queue.head = 0
	queue.tail = 0

	queue.maxSize = maxSize

	queue.nodes = make([]unsafe.Pointer, maxSize)
	return queue
}

func (q *LQueue) IsEmpty() bool {
	return q.head == q.tail
}

func (q *LQueue) IsFull() bool {
	return q.head == (q.tail+1)%q.maxSize
}

func (q *LQueue) IsClosed() bool {
	return q.closed
}

func (q *LQueue) Close() {
	q.lock.Lock()
	q.closed = true
	q.notFull.Broadcast()
	q.notEmpty.Broadcast()
	q.lock.Unlock()
}

func (q *LQueue) Push(p *packet.Packet) error {
	q.lock.Lock()
LOOP:
	if q.IsClosed() {
		q.lock.Unlock()
		return Closed
	}
	if q.IsFull() {
		q.notFull.Wait()
		goto LOOP
	}

	q.nodes[q.tail] = unsafe.Pointer(p)
	q.tail = (q.tail + 1) % q.maxSize
	q.notEmpty.Signal()
	q.lock.Unlock()
	return nil
}

func (q *LQueue) Pop() (p *packet.Packet, err error) {
	q.lock.Lock()
LOOP:
	if q.IsClosed() {
		q.lock.Unlock()
		return nil, Closed
	}
	if q.IsEmpty() {
		q.notEmpty.Wait()
		goto LOOP
	}

	p = (*packet.Packet)(q.nodes[q.head])
	q.head = (q.head + 1) % q.maxSize
	q.notFull.Signal()
	q.lock.Unlock()
	return
}

func (q *LQueue) Drain() (packets []*packet.Packet) {
	q.lock.Lock()
	if q.IsEmpty() {
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

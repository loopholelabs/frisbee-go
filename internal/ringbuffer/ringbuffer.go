/*
	Copyright 2021 Loophole Labs

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

package ringbuffer

import (
	"github.com/loophole-labs/frisbee/internal/errors"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"runtime"
	"sync/atomic"
	"unsafe"
)

var (
	RingerBufferClosed = errors.New("ring buffer is closed")
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

type nodes []node

type RingBuffer struct {
	_padding0    [8]uint64 //nolint:structcheck,unused
	head         uint64
	_padding1    [8]uint64 //nolint:structcheck,unused
	tail         uint64
	_padding2    [8]uint64 //nolint:structcheck,unused
	mask, closed uint64
	_padding3    [8]uint64 //nolint:structcheck,unused
	nodes        nodes
}

func (rb *RingBuffer) init(size uint64) {
	size = round(size)
	rb.nodes = make(nodes, size)
	for i := uint64(0); i < size; i++ {
		rb.nodes[i] = node{position: i}
	}
	rb.mask = size - 1
}

func (rb *RingBuffer) Push(item *protocol.Packet) error {
	var newNode *node
	position := atomic.LoadUint64(&rb.head)
	tail := atomic.LoadUint64(&rb.tail)
	if rb.Capacity() == position-tail {
		atomic.StoreUint64(&rb.head, tail)
		position = tail
	}
RETRY:
	for {
		if atomic.LoadUint64(&rb.closed) == 1 {
			return RingerBufferClosed
		}

		newNode = &rb.nodes[position&rb.mask]
		switch dif := atomic.LoadUint64(&newNode.position) - position; {
		case dif == 0:
			if atomic.CompareAndSwapUint64(&rb.head, position, position+1) {
				break RETRY
			}
		case dif == 1:
			newNode.position -= 1
		default:
			position = atomic.LoadUint64(&rb.head)
		}
		runtime.Gosched()
	}
	newNode.data = unsafe.Pointer(item)
	atomic.StoreUint64(&newNode.position, position+1)
	return nil
}

func (rb *RingBuffer) Pop() (*protocol.Packet, error) {
	var oldNode *node
	var oldPosition = atomic.LoadUint64(&rb.tail)
RETRY:
	for {
		if atomic.LoadUint64(&rb.closed) == 1 {
			return nil, RingerBufferClosed
		}

		oldNode = &rb.nodes[oldPosition&rb.mask]
		switch dif := atomic.LoadUint64(&oldNode.position) - (oldPosition + 1); {
		case dif == 0:
			if atomic.CompareAndSwapUint64(&rb.tail, oldPosition, oldPosition+1) {
				break RETRY
			}
		default:
			oldPosition = atomic.LoadUint64(&rb.tail)
		}
		runtime.Gosched()
	}
	data := oldNode.data
	oldNode.data = nil
	atomic.StoreUint64(&oldNode.position, oldPosition+rb.mask+1)
	return (*protocol.Packet)(data), nil
}

func (rb *RingBuffer) Length() uint64 {
	return atomic.LoadUint64(&rb.head) - atomic.LoadUint64(&rb.tail)
}

func (rb *RingBuffer) Capacity() uint64 {
	return uint64(len(rb.nodes))
}

func (rb *RingBuffer) Close() {
	atomic.CompareAndSwapUint64(&rb.closed, 0, 1)
}

func (rb *RingBuffer) Open() {
	atomic.CompareAndSwapUint64(&rb.closed, 1, 0)
}

func (rb *RingBuffer) IsClosed() bool {
	return atomic.LoadUint64(&rb.closed) == 1
}

func NewRingBuffer(size uint64) *RingBuffer {
	rb := &RingBuffer{}
	if size < 1 {
		size = 1
	}
	rb.init(size)
	return rb
}

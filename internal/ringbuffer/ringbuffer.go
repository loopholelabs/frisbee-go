package ringbuffer

import (
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/pkg/errors"
	"runtime"
	"sync/atomic"
	"unsafe"
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

func (rb *RingBuffer) Push(item *protocol.PacketV0) error {
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
			return errors.New("ring buffer is closed")
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

func (rb *RingBuffer) Pop() (*protocol.PacketV0, error) {
	var oldNode *node
	var oldPosition = atomic.LoadUint64(&rb.tail)
RETRY:
	for {
		if atomic.LoadUint64(&rb.closed) == 1 {
			return nil, errors.New("ring buffer is closed")
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
	return (*protocol.PacketV0)(data), nil
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

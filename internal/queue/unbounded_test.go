package queue

import (
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/stretchr/testify/assert"
	"testing"
	"unsafe"
)

func TestUnbounded(t *testing.T) {
	testPacket := func() unsafe.Pointer {
		return unsafe.Pointer(new(packet.Packet))
	}

	t.Run("success", func(t *testing.T) {
		rb := NewUnbounded()
		p := testPacket()
		err := rb.Push(p)
		assert.NoError(t, err)
		actual, err := rb.Pop()
		assert.NoError(t, err)
		assert.Equal(t, p, actual)
	})
	t.Run("not out of capacity", func(t *testing.T) {
		rb := NewUnbounded()
		err := rb.Push(testPacket())
		assert.NoError(t, err)
	})
	t.Run("buffer closed", func(t *testing.T) {
		rb := NewBounded(1)
		assert.False(t, rb.IsClosed())
		rb.Close()
		assert.True(t, rb.IsClosed())
		err := rb.Push(testPacket())
		assert.ErrorIs(t, Closed, err)
		_, err = rb.Pop()
		assert.ErrorIs(t, Closed, err)
	})
	t.Run("pop empty", func(t *testing.T) {
		done := make(chan struct{}, 1)
		rb := NewBounded(1)
		go func() {
			_, _ = rb.Pop()
			done <- struct{}{}
		}()
		assert.Equal(t, 0, len(done))
		_ = rb.Push(testPacket())
		<-done
		assert.Equal(t, 0, rb.Length())
	})
}

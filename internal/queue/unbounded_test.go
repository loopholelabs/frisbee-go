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

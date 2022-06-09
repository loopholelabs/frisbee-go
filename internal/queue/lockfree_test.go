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
	"testing"
	"time"

	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockFree(t *testing.T) {
	t.Parallel()

	testPacket := packet.Get
	testPacket2 := func() *packet.Packet {
		p := packet.Get()
		p.Content.Write([]byte{1})
		return p
	}

	t.Run("success", func(t *testing.T) {
		rb := NewLockFree[packet.Packet, *packet.Packet](1)
		p := testPacket()
		err := rb.Push(p)
		assert.NoError(t, err)
		actual, err := rb.Pop()
		assert.NoError(t, err)
		assert.Equal(t, p, actual)
	})
	t.Run("out of capacity with non zero capacity, blocking", func(t *testing.T) {
		rb := NewLockFree[packet.Packet, *packet.Packet](1)
		p1 := testPacket()
		err := rb.Push(p1)
		assert.NoError(t, err)
		doneCh := make(chan struct{}, 1)
		p2 := testPacket2()
		go func() {
			err = rb.Push(p2)
			assert.NoError(t, err)
			doneCh <- struct{}{}
		}()
		select {
		case <-doneCh:
			t.Fatal("LockFree did not block on full write")
		case <-time.After(time.Millisecond * 10):
			actual, err := rb.Pop()
			require.NoError(t, err)
			assert.Equal(t, p1, actual)
			select {
			case <-doneCh:
				actual, err := rb.Pop()
				require.NoError(t, err)
				assert.Equal(t, p2, actual)
			case <-time.After(time.Millisecond * 10):
				t.Fatal("LockFree did not unblock on read from full write")
			}
		}
	})
	t.Run("buffer closed", func(t *testing.T) {
		rb := NewLockFree[packet.Packet, *packet.Packet](1)
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
		rb := NewLockFree[packet.Packet, *packet.Packet](1)
		go func() {
			_, _ = rb.Pop()
			done <- struct{}{}
		}()
		assert.Equal(t, 0, len(done))
		_ = rb.Push(testPacket())
		<-done
		assert.Equal(t, 0, rb.Length())
	})
	t.Run("partial overflow, blocking", func(t *testing.T) {
		rb := NewLockFree[packet.Packet, *packet.Packet](4)
		p1 := testPacket()
		p1.Metadata.Id = 1

		p2 := testPacket()
		p2.Metadata.Id = 2

		p3 := testPacket()
		p3.Metadata.Id = 3

		p4 := testPacket()
		p4.Metadata.Id = 4

		p5 := testPacket()
		p5.Metadata.Id = 5

		err := rb.Push(p1)
		assert.NoError(t, err)
		err = rb.Push(p2)
		assert.NoError(t, err)
		err = rb.Push(p3)
		assert.NoError(t, err)

		assert.Equal(t, 3, rb.Length())

		actual, err := rb.Pop()
		assert.NoError(t, err)
		assert.Equal(t, p1, actual)
		assert.Equal(t, 2, rb.Length())

		err = rb.Push(p4)
		assert.NoError(t, err)
		err = rb.Push(p5)
		assert.NoError(t, err)

		assert.Equal(t, 4, rb.Length())

		actual, err = rb.Pop()
		assert.NoError(t, err)
		assert.Equal(t, p2, actual)

		assert.Equal(t, 3, rb.Length())

		actual, err = rb.Pop()
		assert.NoError(t, err)
		assert.Equal(t, p3, actual)

		assert.Equal(t, 2, rb.Length())

		actual, err = rb.Pop()
		assert.NoError(t, err)
		assert.Equal(t, p4, actual)

		assert.Equal(t, 1, rb.Length())

		actual, err = rb.Pop()
		assert.NoError(t, err)
		assert.Equal(t, p5, actual)
		assert.NotEqual(t, p1, p5)
		assert.Equal(t, 0, rb.Length())
	})
}

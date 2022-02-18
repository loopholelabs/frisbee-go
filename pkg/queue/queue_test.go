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
	"github.com/stretchr/testify/require"
	"testing"
	"time"
	"unsafe"
)

func TestBoundedHelpers(t *testing.T) {
	t.Run("test round", func(t *testing.T) {
		tcs := []struct {
			in       uint64
			expected uint64
		}{
			{in: 0, expected: 0x0},
			{in: 1, expected: 0x1},
			{in: 2, expected: 0x2},
			{in: 3, expected: 0x4},
			{in: 4, expected: 0x4},
			{in: 5, expected: 0x8},
			{in: 7, expected: 0x8},
			{in: 8, expected: 0x8},
			{in: 9, expected: 0x10},
			{in: 16, expected: 0x10},
			{in: 32, expected: 0x20},
			{in: 0xFFFFFFF0, expected: 0x100000000},
			{in: 0xFFFFFFFF, expected: 0x100000000},
		}
		for _, tc := range tcs {
			assert.Equalf(t, tc.expected, round(tc.in), "in: %d", tc.in)
		}
	})
}

func TestBounded(t *testing.T) {
	testPacket := func() unsafe.Pointer {
		return unsafe.Pointer(new(packet.Packet))
	}
	testPacket2 := func() unsafe.Pointer {
		return unsafe.Pointer(&packet.Packet{
			Content: []byte{1},
		})
	}

	t.Run("success", func(t *testing.T) {
		rb := New(1, false)
		p := testPacket()
		err := rb.Push(p)
		assert.NoError(t, err)
		actual, err := rb.Pop()
		assert.NoError(t, err)
		assert.Equal(t, p, actual)
	})
	t.Run("out of capacity", func(t *testing.T) {
		rb := New(0, false)
		err := rb.Push(testPacket())
		assert.NoError(t, err)
	})
	t.Run("out of capacity with non zero capacity", func(t *testing.T) {
		rb := New(1, true)
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
			t.Fatal("Queue did not block on full write")
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
				t.Fatal("Queue did not unblock on read from full write")
			}
		}

	})
	t.Run("buffer closed", func(t *testing.T) {
		rb := New(1, false)
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
		rb := New(1, false)
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

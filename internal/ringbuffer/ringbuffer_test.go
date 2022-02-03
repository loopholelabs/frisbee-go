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

package ringbuffer

import (
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHelpers(t *testing.T) {
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

func TestRingBuffer(t *testing.T) {
	testPacket := func() *packet.Packet {
		return &packet.Packet{}
	}
	testPacket2 := func() *packet.Packet {
		return &packet.Packet{
			Content: []byte{1},
		}
	}

	t.Run("success", func(t *testing.T) {
		rb := NewRingBuffer(1)
		_ = rb.Push(testPacket())
		actual, err := rb.Pop()
		assert.NoError(t, err)
		assert.Equal(t, testPacket(), actual)
	})
	t.Run("out of capacity", func(t *testing.T) {
		rb := NewRingBuffer(0)
		err := rb.Push(testPacket())
		assert.NoError(t, err)
	})
	t.Run("out of capacity with non zero capacity", func(t *testing.T) {
		rb := NewRingBuffer(1)
		err := rb.Push(testPacket())
		assert.NoError(t, err)
		err = rb.Push(testPacket2())
		assert.NoError(t, err)
		actual, err := rb.Pop()
		require.NoError(t, err)
		assert.Equal(t, testPacket2(), actual)
	})
	t.Run("buffer closed", func(t *testing.T) {
		rb := NewRingBuffer(1)
		assert.False(t, rb.IsClosed())
		rb.Close()
		assert.True(t, rb.IsClosed())
		err := rb.Push(testPacket())
		assert.ErrorIs(t, RingerBufferClosed, err)
		_, err = rb.Pop()
		assert.ErrorIs(t, RingerBufferClosed, err)
	})
	t.Run("pop empty", func(t *testing.T) {
		done := make(chan struct{}, 1)
		rb := NewRingBuffer(1)
		go func() {
			_, _ = rb.Pop()
			done <- struct{}{}
		}()
		assert.Equal(t, 0, len(done))
		_ = rb.Push(testPacket())
		<-done
		assert.Equal(t, uint64(0), rb.Length())
	})
	t.Run("buffer closed then reopened", func(t *testing.T) {
		rb := NewRingBuffer(1)
		assert.False(t, rb.IsClosed())
		rb.Close()
		assert.True(t, rb.IsClosed())
		err := rb.Push(testPacket())
		assert.ErrorIs(t, RingerBufferClosed, err)
		_, err = rb.Pop()
		assert.ErrorIs(t, RingerBufferClosed, err)
		rb.Open()
		assert.False(t, rb.IsClosed())
		err = rb.Push(testPacket())
		assert.NoError(t, err)
		actual, err := rb.Pop()
		assert.NoError(t, err)
		assert.Equal(t, testPacket(), actual)
	})
}

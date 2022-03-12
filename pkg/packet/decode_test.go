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

package packet

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDecodeNil(t *testing.T) {
	t.Parallel()

	p := Get()
	encodeNil(p)

	var value bool

	remaining, value := decodeNil(p.Content.B)
	assert.True(t, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, value = decodeNil(p.Content.B)
	assert.False(t, value)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value = decodeNil(p.Content.B)
	assert.True(t, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.True(t, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeNil(p)
		remaining, value = decodeNil(p.Content.B)
		p.Content.Reset()
	})
	assert.Zero(t, n)
	Put(p)
}

func TestDecodeMap(t *testing.T) {
	t.Parallel()

	p := Get()
	encodeMap(p, 32, StringKind, Uint32Kind)

	remaining, size, err := decodeMap(p.Content.B, StringKind, Uint32Kind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(32), size)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeMap(p.Content.B, StringKind, Uint32Kind)
	assert.ErrorIs(t, err, InvalidMap)

	p.Content.B = append([]byte{b}, p.Content.B...)
	_, _, err = decodeMap(p.Content.B, StringKind, Float64Kind)
	assert.ErrorIs(t, err, InvalidMap)

	remaining, size, err = decodeMap(p.Content.B, StringKind, Uint32Kind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(32), size)
	assert.Equal(t, 0, len(remaining))

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeNil(p)
		remaining, size, err = decodeMap(p.Content.B, StringKind, Uint32Kind)
		p.Content.Reset()
	})
	assert.Zero(t, n)
	Put(p)
}

func TestDecodeBytes(t *testing.T) {
	t.Parallel()

	p := Get()
	v := []byte("Test Bytes")
	encodeBytes(p, v)

	var value []byte

	remaining, value, err := decodeBytes(p.Content.B, value)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, value, err = decodeBytes(p.Content.B, value)
	assert.ErrorIs(t, err, InvalidBytes)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeBytes(p.Content.B, value)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.Equal(t, v, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeBytes(p, v)
		remaining, value, err = decodeBytes(p.Content.B, value)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	n = testing.AllocsPerRun(100, func() {
		encodeBytes(p, v)
		remaining, value, err = decodeBytes(p.Content.B, nil)
		p.Content.Reset()
	})
	assert.Equal(t, float64(1), n)

	s := [][]byte{v, v, v, v, v}
	encodeSlice(p, uint32(len(s)), BytesKind)
	for _, sb := range s {
		encodeBytes(p, sb)
	}
	var size uint32

	remaining, size, err = decodeSlice(p.Content.B, BytesKind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(s)), size)

	sValue := make([][]byte, size)
	for i := uint32(0); i < size; i++ {
		remaining, sValue[i], err = decodeBytes(remaining, nil)
		assert.NoError(t, err)
		assert.Equal(t, s[i], sValue[i])
	}

	assert.Equal(t, s, sValue)
	assert.Equal(t, 0, len(remaining))

	Put(p)
}

func TestDecodeString(t *testing.T) {
	t.Parallel()

	p := Get()
	v := "Test String"
	encodeString(p, v)

	var value string

	remaining, value, err := decodeString(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeString(p.Content.B)
	assert.ErrorIs(t, err, InvalidString)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeString(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.Equal(t, v, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeString(p, v)
		remaining, value, err = decodeString(p.Content.B)
		p.Content.Reset()
	})
	assert.Equal(t, float64(1), n)

	s := []string{v, v, v, v, v}
	encodeSlice(p, uint32(len(s)), StringKind)
	for _, sb := range s {
		encodeString(p, sb)
	}
	var size uint32

	remaining, size, err = decodeSlice(p.Content.B, StringKind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(s)), size)

	sValue := make([]string, size)
	for i := uint32(0); i < size; i++ {
		remaining, sValue[i], err = decodeString(remaining)
		assert.NoError(t, err)
		assert.Equal(t, s[i], sValue[i])
	}

	assert.Equal(t, s, sValue)
	assert.Equal(t, 0, len(remaining))

	Put(p)
}

func TestDecodeError(t *testing.T) {
	t.Parallel()

	p := Get()
	v := errors.New("Test Error")
	encodeError(p, v)

	var value error

	remaining, value, err := decodeError(p.Content.B)
	assert.NoError(t, err)
	assert.ErrorIs(t, value, v)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeError(p.Content.B)
	assert.ErrorIs(t, err, InvalidError)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeError(p.Content.B)
	assert.NoError(t, err)
	assert.ErrorIs(t, value, v)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.ErrorIs(t, value, v)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeError(p, v)
		remaining, value, err = decodeError(p.Content.B)
		p.Content.Reset()
	})
	assert.Equal(t, float64(2), n)

	s := []error{v, v, v, v, v}
	encodeSlice(p, uint32(len(s)), ErrorKind)
	for _, sb := range s {
		encodeError(p, sb)
	}
	var size uint32

	remaining, size, err = decodeSlice(p.Content.B, ErrorKind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(s)), size)

	sValue := make([]error, size)
	for i := uint32(0); i < size; i++ {
		remaining, sValue[i], err = decodeError(remaining)
		assert.NoError(t, err)
		assert.ErrorIs(t, sValue[i], s[i])
	}

	assert.Equal(t, 0, len(remaining))

	Put(p)
}

func TestDecodeBool(t *testing.T) {
	t.Parallel()

	p := Get()
	encodeBool(p, true)

	var value bool

	remaining, value, err := decodeBool(p.Content.B)
	assert.NoError(t, err)
	assert.True(t, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeBool(p.Content.B)
	assert.ErrorIs(t, err, InvalidBool)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeBool(p.Content.B)
	assert.NoError(t, err)
	assert.True(t, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.True(t, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeBool(p, true)
		remaining, value, err = decodeBool(p.Content.B)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	s := []bool{true, true, false, true, true}
	encodeSlice(p, uint32(len(s)), BoolKind)
	for _, sb := range s {
		encodeBool(p, sb)
	}
	var size uint32

	remaining, size, err = decodeSlice(p.Content.B, BoolKind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(s)), size)

	sValue := make([]bool, size)
	for i := uint32(0); i < size; i++ {
		remaining, sValue[i], err = decodeBool(remaining)
		assert.NoError(t, err)
		assert.Equal(t, s[i], sValue[i])
	}

	assert.Equal(t, s, sValue)
	assert.Equal(t, 0, len(remaining))

	Put(p)
}

func TestDecodeUint8(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint8(32)
	encodeUint8(p, v)

	var value uint8

	remaining, value, err := decodeUint8(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeUint8(p.Content.B)
	assert.ErrorIs(t, err, InvalidUint8)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeUint8(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.Equal(t, v, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeUint8(p, v)
		remaining, value, err = decodeUint8(p.Content.B)
		p.Content.Reset()
	})
	assert.Zero(t, n)
	Put(p)
}

func TestDecodeUint16(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint16(1024)
	encodeUint16(p, v)

	var value uint16

	remaining, value, err := decodeUint16(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeUint16(p.Content.B)
	assert.ErrorIs(t, err, InvalidUint16)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeUint16(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.Equal(t, v, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeUint16(p, v)
		remaining, value, err = decodeUint16(p.Content.B)
		p.Content.Reset()
	})
	assert.Zero(t, n)
	Put(p)
}

func TestDecodeUint32(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint32(4294967290)
	encodeUint32(p, v)

	var value uint32

	remaining, value, err := decodeUint32(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeUint32(p.Content.B)
	assert.ErrorIs(t, err, InvalidUint32)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeUint32(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.Equal(t, v, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeUint32(p, v)
		remaining, value, err = decodeUint32(p.Content.B)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecodeUint64(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint64(18446744073709551610)
	encodeUint64(p, v)

	var value uint64

	remaining, value, err := decodeUint64(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeUint64(p.Content.B)
	assert.ErrorIs(t, err, InvalidUint64)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeUint64(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.Equal(t, v, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeUint64(p, v)
		remaining, value, err = decodeUint64(p.Content.B)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecodeInt32(t *testing.T) {
	t.Parallel()

	p := Get()
	v := int32(-2147483648)
	encodeInt32(p, v)

	var value int32

	remaining, value, err := decodeInt32(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeInt32(p.Content.B)
	assert.ErrorIs(t, err, InvalidInt32)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeInt32(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.Equal(t, v, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeInt32(p, v)
		remaining, value, err = decodeInt32(p.Content.B)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecodeInt64(t *testing.T) {
	t.Parallel()

	p := Get()
	v := int64(-9223372036854775808)
	encodeInt64(p, v)

	var value int64

	remaining, value, err := decodeInt64(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeInt64(p.Content.B)
	assert.ErrorIs(t, err, InvalidInt64)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeInt64(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.Equal(t, v, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeInt64(p, v)
		remaining, value, err = decodeInt64(p.Content.B)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecodeFloat32(t *testing.T) {
	t.Parallel()

	p := Get()
	v := float32(-12311.12429)
	encodeFloat32(p, v)

	var value float32

	remaining, value, err := decodeFloat32(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeFloat32(p.Content.B)
	assert.ErrorIs(t, err, InvalidFloat32)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeFloat32(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.Equal(t, v, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeFloat32(p, v)
		remaining, value, err = decodeFloat32(p.Content.B)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecodeFloat64(t *testing.T) {
	t.Parallel()

	p := Get()
	v := -12311241.1242009
	encodeFloat64(p, v)

	var value float64

	remaining, value, err := decodeFloat64(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	var b byte
	b, p.Content.B = p.Content.B[0], p.Content.B[1:]
	_, _, err = decodeFloat64(p.Content.B)
	assert.ErrorIs(t, err, InvalidFloat64)

	p.Content.B = append([]byte{b}, p.Content.B...)
	remaining, value, err = decodeFloat64(p.Content.B)
	assert.NoError(t, err)
	assert.Equal(t, v, value)
	assert.Equal(t, 0, len(remaining))

	p.Content.B[len(p.Content.B)-1] = 'S'
	assert.Equal(t, v, value)

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeFloat64(p, v)
		remaining, value, err = decodeFloat64(p.Content.B)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

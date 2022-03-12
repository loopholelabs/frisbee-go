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
	"math"
	"testing"
)

func TestEncodeNil(t *testing.T) {
	t.Parallel()

	p := Get()
	encodeNil(p)

	assert.Equal(t, 1, len(p.Content.B))
	assert.Equal(t, NilKind, Kind(p.Content.B))

	n := testing.AllocsPerRun(100, func() {
		encodeNil(p)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeMap(t *testing.T) {
	t.Parallel()

	p := Get()
	encodeMap(p, 32, StringKind, Uint32Kind)

	assert.Equal(t, 1+1+1+1+4, len(p.Content.B))
	assert.Equal(t, MapKind, Kind(p.Content.B[0:1]))
	assert.Equal(t, StringKind, Kind(p.Content.B[1:2]))
	assert.Equal(t, Uint32Kind, Kind(p.Content.B[2:3]))
	assert.Equal(t, Uint32Kind, Kind(p.Content.B[3:4]))

	n := testing.AllocsPerRun(100, func() {
		encodeMap(p, 32, StringKind, Uint32Kind)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeSlice(t *testing.T) {
	t.Parallel()

	p := Get()
	encodeSlice(p, 32, StringKind)

	assert.Equal(t, 1+1+1+4, len(p.Content.B))
	assert.Equal(t, SliceKind, Kind(p.Content.B[0:1]))
	assert.Equal(t, StringKind, Kind(p.Content.B[1:2]))
	assert.Equal(t, Uint32Kind, Kind(p.Content.B[2:3]))

	n := testing.AllocsPerRun(100, func() {
		encodeSlice(p, 32, StringKind)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeBytes(t *testing.T) {
	t.Parallel()

	p := Get()
	v := []byte("Test String")

	encodeBytes(p, v)

	assert.Equal(t, 1+1+4+len(v), len(p.Content.B))
	assert.Equal(t, v, p.Content.B[1+1+4:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeBytes(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)
	Put(p)
}

func TestEncodeString(t *testing.T) {
	t.Parallel()

	p := Get()
	v := "Test String"
	e := []byte(v)

	encodeString(p, v)

	assert.Equal(t, 1+1+4+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1+1+4:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeString(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeError(t *testing.T) {
	t.Parallel()

	p := Get()
	v := errors.New("Test Error")
	e := []byte(v.Error())

	encodeError(p, v)

	assert.Equal(t, 1+1+1+4+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1+1+1+4:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeError(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeBool(t *testing.T) {
	t.Parallel()

	p := Get()
	e := []byte{trueBool}

	encodeBool(p, true)

	assert.Equal(t, 1+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeBool(p, true)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeUint8(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint8(32)
	e := []byte{v}

	encodeUint8(p, v)

	assert.Equal(t, 1+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeUint8(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeUint16(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint16(1024)
	e := []byte{byte(v >> 8), byte(v)}

	encodeUint16(p, v)

	assert.Equal(t, 1+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeUint16(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeUint32(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint32(4294967290)
	e := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}

	encodeUint32(p, v)

	assert.Equal(t, 1+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeUint32(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeUint64(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint64(18446744073709551610)
	e := []byte{byte(v >> 56), byte(v >> 48), byte(v >> 40), byte(v >> 32), byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}

	encodeUint64(p, v)

	assert.Equal(t, 1+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeUint64(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeInt32(t *testing.T) {
	t.Parallel()

	p := Get()
	v := int32(-2147483648)
	e := []byte{byte(uint32(v) >> 24), byte(uint32(v) >> 16), byte(uint32(v) >> 8), byte(uint32(v))}

	encodeInt32(p, v)

	assert.Equal(t, 1+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeInt32(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeInt64(t *testing.T) {
	t.Parallel()

	p := Get()
	v := int64(-9223372036854775808)
	e := []byte{byte(uint64(v) >> 56), byte(uint64(v) >> 48), byte(uint64(v) >> 40), byte(uint64(v) >> 32), byte(uint64(v) >> 24), byte(uint64(v) >> 16), byte(uint64(v) >> 8), byte(uint64(v))}

	encodeInt64(p, v)

	assert.Equal(t, 1+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeInt64(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeFloat32(t *testing.T) {
	t.Parallel()

	p := Get()
	v := float32(-214648.34432)
	e := []byte{byte(math.Float32bits(v) >> 24), byte(math.Float32bits(v) >> 16), byte(math.Float32bits(v) >> 8), byte(math.Float32bits(v))}

	encodeFloat32(p, v)

	assert.Equal(t, 1+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeFloat32(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestEncodeFloat64(t *testing.T) {
	t.Parallel()

	p := Get()
	v := -922337203685.2345
	e := []byte{byte(math.Float64bits(v) >> 56), byte(math.Float64bits(v) >> 48), byte(math.Float64bits(v) >> 40), byte(math.Float64bits(v) >> 32), byte(math.Float64bits(v) >> 24), byte(math.Float64bits(v) >> 16), byte(math.Float64bits(v) >> 8), byte(math.Float64bits(v))}

	encodeFloat64(p, v)

	assert.Equal(t, 1+len(e), len(p.Content.B))
	assert.Equal(t, e, p.Content.B[1:])

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeFloat64(p, v)
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

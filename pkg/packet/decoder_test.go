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

func TestDecoderNil(t *testing.T) {
	t.Parallel()

	p := Get()
	Encoder(p).Nil()

	d := GetDecoder(p.Content.B)
	value := d.Nil()
	assert.True(t, value)

	value = d.Nil()
	assert.False(t, value)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Nil()
		d = GetDecoder(p.Content.B)
		value = d.Nil()
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderMap(t *testing.T) {
	t.Parallel()

	p := Get()
	m := make(map[string]uint32)
	m["1"] = 1
	m["2"] = 2
	m["3"] = 3

	e := Encoder(p).Map(uint32(len(m)), StringKind, Uint32Kind)
	for k, v := range m {
		e.String(k).Uint32(v)
	}

	d := GetDecoder(p.Content.B)
	size, err := d.Map(StringKind, Uint32Kind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(m)), size)

	mv := make(map[string]uint32, size)
	var k string
	var v uint32
	for i := uint32(0); i < size; i++ {
		k, err = d.String()
		assert.NoError(t, err)
		v, err = d.Uint32()
		assert.NoError(t, err)
		mv[k] = v
	}
	assert.Equal(t, m, mv)

	size, err = d.Map(StringKind, Uint32Kind)
	assert.ErrorIs(t, err, InvalidMap)
	assert.Equal(t, uint32(0), size)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		e = Encoder(p).Map(uint32(len(m)), StringKind, Uint32Kind)
		for k, v = range m {
			e.String(k).Uint32(v)
		}
		d = GetDecoder(p.Content.B)
		size, err = d.Map(StringKind, Uint32Kind)
		for i := uint32(0); i < size; i++ {
			_, _ = d.String()
			_, _ = d.Uint32()
		}
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderSlice(t *testing.T) {
	t.Parallel()

	p := Get()
	m := []string{"1", "2", "3"}

	e := Encoder(p).Slice(uint32(len(m)), StringKind)
	for _, v := range m {
		e.String(v)
	}

	d := GetDecoder(p.Content.B)
	size, err := d.Slice(StringKind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(m)), size)

	mv := make([]string, size)
	for i := range mv {
		mv[i], err = d.String()
		assert.NoError(t, err)
		assert.Equal(t, m[i], mv[i])
	}
	assert.Equal(t, m, mv)

	size, err = d.Slice(StringKind)
	assert.ErrorIs(t, err, InvalidSlice)
	assert.Equal(t, uint32(0), size)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		e = Encoder(p).Slice(uint32(len(m)), StringKind)
		for _, v := range m {
			e.String(v)
		}
		d = GetDecoder(p.Content.B)
		size, err = d.Slice(StringKind)
		for i := uint32(0); i < size; i++ {
			_, _ = d.String()
		}
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderBytes(t *testing.T) {
	t.Parallel()

	p := Get()
	v := []byte("Test String")

	Encoder(p).Bytes(v)

	d := GetDecoder(p.Content.B)
	value, err := d.Bytes(nil)
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	value, err = d.Bytes(value)
	assert.ErrorIs(t, err, InvalidBytes)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Bytes(v)
		d = GetDecoder(p.Content.B)
		value, err = d.Bytes(value)
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderString(t *testing.T) {
	t.Parallel()

	p := Get()
	v := "Test String"

	Encoder(p).String(v)

	d := GetDecoder(p.Content.B)
	value, err := d.String()
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	value, err = d.String()
	assert.ErrorIs(t, err, InvalidString)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).String(v)
		d = GetDecoder(p.Content.B)
		value, err = d.String()
		d.Return()
		p.Content.Reset()
	})
	assert.Equal(t, float64(1), n)

	Put(p)
}

func TestDecoderError(t *testing.T) {
	t.Parallel()

	p := Get()
	v := errors.New("Test Error")

	Encoder(p).Error(v)

	d := GetDecoder(p.Content.B)
	value, err := d.Error()
	assert.NoError(t, err)
	assert.ErrorIs(t, value, v)

	value, err = d.Error()
	assert.ErrorIs(t, err, InvalidError)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Error(v)
		d = GetDecoder(p.Content.B)
		value, err = d.Error()
		d.Return()
		p.Content.Reset()
	})
	assert.Equal(t, float64(2), n)

	Put(p)
}

func TestDecoderBool(t *testing.T) {
	t.Parallel()

	p := Get()
	Encoder(p).Bool(true)

	d := GetDecoder(p.Content.B)
	value, err := d.Bool()
	assert.NoError(t, err)
	assert.True(t, value)

	value, err = d.Bool()
	assert.ErrorIs(t, err, InvalidBool)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Bool(true)
		d = GetDecoder(p.Content.B)
		value, err = d.Bool()
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderUint8(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint8(32)

	Encoder(p).Uint8(v)

	d := GetDecoder(p.Content.B)
	value, err := d.Uint8()
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	value, err = d.Uint8()
	assert.ErrorIs(t, err, InvalidUint8)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Uint8(v)
		d = GetDecoder(p.Content.B)
		value, err = d.Uint8()
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderUint16(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint16(1024)

	Encoder(p).Uint16(v)

	d := GetDecoder(p.Content.B)
	value, err := d.Uint16()
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	value, err = d.Uint16()
	assert.ErrorIs(t, err, InvalidUint16)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Uint16(v)
		d = GetDecoder(p.Content.B)
		value, err = d.Uint16()
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderUint32(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint32(4294967290)

	Encoder(p).Uint32(v)

	d := GetDecoder(p.Content.B)
	value, err := d.Uint32()
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	value, err = d.Uint32()
	assert.ErrorIs(t, err, InvalidUint32)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Uint32(v)
		d = GetDecoder(p.Content.B)
		value, err = d.Uint32()
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderUint64(t *testing.T) {
	t.Parallel()

	p := Get()
	v := uint64(18446744073709551610)

	Encoder(p).Uint64(v)

	d := GetDecoder(p.Content.B)
	value, err := d.Uint64()
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	value, err = d.Uint64()
	assert.ErrorIs(t, err, InvalidUint64)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Uint64(v)
		d = GetDecoder(p.Content.B)
		value, err = d.Uint64()
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderInt32(t *testing.T) {
	t.Parallel()

	p := Get()
	v := int32(-2147483648)

	Encoder(p).Int32(v)

	d := GetDecoder(p.Content.B)
	value, err := d.Int32()
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	value, err = d.Int32()
	assert.ErrorIs(t, err, InvalidInt32)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Int32(v)
		d = GetDecoder(p.Content.B)
		value, err = d.Int32()
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderInt64(t *testing.T) {
	t.Parallel()

	p := Get()
	v := int64(-9223372036854775808)

	Encoder(p).Int64(v)

	d := GetDecoder(p.Content.B)
	value, err := d.Int64()
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	value, err = d.Int64()
	assert.ErrorIs(t, err, InvalidInt64)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Int64(v)
		d = GetDecoder(p.Content.B)
		value, err = d.Int64()
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderFloat32(t *testing.T) {
	t.Parallel()

	p := Get()
	v := float32(-2147483.648)

	Encoder(p).Float32(v)

	d := GetDecoder(p.Content.B)
	value, err := d.Float32()
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	value, err = d.Float32()
	assert.ErrorIs(t, err, InvalidFloat32)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Float32(v)
		d = GetDecoder(p.Content.B)
		value, err = d.Float32()
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

func TestDecoderFloat64(t *testing.T) {
	t.Parallel()

	p := Get()
	v := -922337203.477580

	Encoder(p).Float64(v)

	d := GetDecoder(p.Content.B)
	value, err := d.Float64()
	assert.NoError(t, err)
	assert.Equal(t, v, value)

	value, err = d.Float64()
	assert.ErrorIs(t, err, InvalidFloat64)

	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Float64(v)
		d = GetDecoder(p.Content.B)
		value, err = d.Float64()
		d.Return()
		p.Content.Reset()
	})
	assert.Zero(t, n)

	Put(p)
}

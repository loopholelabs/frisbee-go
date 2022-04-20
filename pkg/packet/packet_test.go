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
	"crypto/rand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	p := Get()

	assert.IsType(t, new(Packet), p)
	assert.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(0), p.Metadata.Id)
	assert.Equal(t, uint16(0), p.Metadata.Operation)
	assert.Equal(t, uint32(0), p.Metadata.ContentLength)
	assert.Equal(t, []byte{}, p.Content.B)

	Put(p)
}

func TestWrite(t *testing.T) {
	t.Parallel()

	p := Get()

	b := make([]byte, 32)
	_, err := rand.Read(b)
	assert.NoError(t, err)

	p.Content.Write(b)
	assert.Equal(t, b, p.Content.B)

	p.Reset()
	assert.NotEqual(t, b, p.Content.B)
	assert.Equal(t, 0, len(p.Content.B))
	assert.Equal(t, 512, cap(p.Content.B))

	b = make([]byte, 1024)
	_, err = rand.Read(b)
	assert.NoError(t, err)

	p.Content.Write(b)

	assert.Equal(t, b, p.Content.B)
	assert.Equal(t, 1024, len(p.Content.B))
	assert.GreaterOrEqual(t, cap(p.Content.B), 1024)

}

type embedStruct struct {
	test string
	b    []byte
}

type testStruct struct {
	err   error
	test  string
	b     []byte
	num1  uint8
	num2  uint16
	num3  uint32
	num4  uint64
	num5  int32
	num6  int64
	num7  float32
	num8  float64
	truth bool
	slice []string
	m     map[uint32]*embedStruct
}

func TestChain(t *testing.T) {
	t.Parallel()

	test := &testStruct{
		err:   errors.New("Test Error"),
		test:  "Test String",
		b:     []byte("Test Bytes"),
		num1:  32,
		num2:  1024,
		num3:  4294967290,
		num4:  18446744073709551610,
		num5:  -123531252,
		num6:  -123514361905132059,
		num7:  -21239.343,
		num8:  -129403505932.823,
		truth: true,
		slice: []string{"te", "s", "t"},
		m:     make(map[uint32]*embedStruct),
	}

	test.m[0] = &embedStruct{
		test: "Embed String",
		b:    []byte("embed Bytes"),
	}

	test.m[1] = &embedStruct{
		test: "Other Embed String",
		b:    []byte("other embed Bytes"),
	}

	p := Get()
	encodeError(p, test.err)
	encodeString(p, test.test)
	encodeBytes(p, test.b)
	encodeUint8(p, test.num1)
	encodeUint16(p, test.num2)
	encodeUint32(p, test.num3)
	encodeUint64(p, test.num4)
	encodeInt32(p, test.num5)
	encodeInt64(p, test.num6)
	encodeFloat32(p, test.num7)
	encodeFloat64(p, test.num8)
	encodeBool(p, test.truth)
	encodeSlice(p, uint32(len(test.slice)), StringKind)
	for _, s := range test.slice {
		encodeString(p, s)
	}
	encodeMap(p, uint32(len(test.m)), Uint32Kind, AnyKind)
	for k, v := range test.m {
		encodeUint32(p, k)
		encodeString(p, v.test)
		encodeBytes(p, v.b)
	}
	encodeNil(p)

	val := new(testStruct)
	var err error
	var remaining []byte

	remaining, val.err, err = decodeError(p.Content.B)
	assert.NoError(t, err)
	assert.ErrorIs(t, val.err, test.err)

	remaining, val.test, err = decodeString(remaining)
	assert.NoError(t, err)
	assert.Equal(t, test.test, val.test)

	remaining, val.b, err = decodeBytes(remaining, nil)
	assert.NoError(t, err)
	assert.Equal(t, test.b, val.b)

	remaining, val.num1, err = decodeUint8(remaining)
	assert.NoError(t, err)
	assert.Equal(t, val.num1, val.num1)

	remaining, val.num2, err = decodeUint16(remaining)
	assert.NoError(t, err)
	assert.Equal(t, test.num2, val.num2)

	remaining, val.num3, err = decodeUint32(remaining)
	assert.NoError(t, err)
	assert.Equal(t, test.num3, val.num3)

	remaining, val.num4, err = decodeUint64(remaining)
	assert.NoError(t, err)
	assert.Equal(t, test.num4, val.num4)

	remaining, val.num5, err = decodeInt32(remaining)
	assert.NoError(t, err)
	assert.Equal(t, test.num5, val.num5)

	remaining, val.num6, err = decodeInt64(remaining)
	assert.NoError(t, err)
	assert.Equal(t, test.num6, val.num6)

	remaining, val.num7, err = decodeFloat32(remaining)
	assert.NoError(t, err)
	assert.Equal(t, test.num7, val.num7)

	remaining, val.num8, err = decodeFloat64(remaining)
	assert.NoError(t, err)
	assert.Equal(t, test.num8, val.num8)

	remaining, val.truth, err = decodeBool(remaining)
	assert.NoError(t, err)
	assert.Equal(t, test.truth, val.truth)

	var size uint32
	remaining, size, err = decodeSlice(remaining, StringKind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(test.slice)), size)

	val.slice = make([]string, size)
	for i := range val.slice {
		remaining, val.slice[i], err = decodeString(remaining)
		assert.NoError(t, err)
		assert.Equal(t, test.slice[i], val.slice[i])
	}
	assert.Equal(t, test.slice, val.slice)

	remaining, size, err = decodeMap(remaining, Uint32Kind, AnyKind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(test.m)), size)
	val.m = make(map[uint32]*embedStruct, size)
	k := uint32(0)
	v := new(embedStruct)
	for i := uint32(0); i < size; i++ {
		remaining, k, err = decodeUint32(remaining)
		assert.NoError(t, err)
		remaining, v.test, err = decodeString(remaining)
		assert.NoError(t, err)
		remaining, v.b, err = decodeBytes(remaining, v.b)
		assert.NoError(t, err)
		val.m[k] = v
		v = new(embedStruct)
	}
	assert.Equal(t, test.m, val.m)

	var isNil bool
	remaining, isNil = decodeNil(remaining)
	assert.True(t, isNil)

	assert.Equal(t, 0, len(remaining))

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		encodeError(p, test.err)
		encodeString(p, test.test)
		encodeBytes(p, test.b)
		encodeUint8(p, test.num1)
		encodeUint16(p, test.num2)
		encodeUint32(p, test.num3)
		encodeUint64(p, test.num4)
		encodeBool(p, test.truth)
		encodeNil(p)
		remaining, val.err, err = decodeError(p.Content.B)
		remaining, val.test, err = decodeString(remaining)
		remaining, val.b, err = decodeBytes(remaining, val.b)
		remaining, val.num1, err = decodeUint8(remaining)
		remaining, val.num2, err = decodeUint16(remaining)
		remaining, val.num3, err = decodeUint32(remaining)
		remaining, val.num4, err = decodeUint64(remaining)
		remaining, val.truth, err = decodeBool(remaining)
		remaining, isNil = decodeNil(remaining)
		p.Content.Reset()
	})
	assert.Equal(t, float64(3), n)
	Put(p)
}

func TestCompleteChain(t *testing.T) {
	t.Parallel()

	test := &testStruct{
		err:   errors.New("Test Error"),
		test:  "Test String",
		b:     []byte("Test Bytes"),
		num1:  32,
		num2:  1024,
		num3:  4294967290,
		num4:  18446744073709551610,
		num5:  -123531252,
		num6:  -123514361905132059,
		num7:  -21239.343,
		num8:  -129403505932.823,
		truth: true,
		slice: []string{"test1", "test2"},
		m:     make(map[uint32]*embedStruct),
	}

	test.m[0] = &embedStruct{
		test: "Embed String",
		b:    []byte("embed Bytes"),
	}

	test.m[1] = &embedStruct{
		test: "Other Embed String",
		b:    []byte("other embed Bytes"),
	}

	p := Get()
	e := Encoder(p).Error(test.err).String(test.test).Bytes(test.b).Uint8(test.num1).Uint16(test.num2).Uint32(test.num3).Uint64(test.num4).Int32(test.num5).Int64(test.num6).Float32(test.num7).Float64(test.num8).Bool(test.truth).Nil().Slice(uint32(len(test.slice)), StringKind)
	for _, s := range test.slice {
		e.String(s)
	}
	e.Map(uint32(len(test.m)), Uint32Kind, AnyKind)
	for k, v := range test.m {
		e.Uint32(k).String(v.test).Bytes(v.b)
	}

	val := new(testStruct)
	var err error

	d := GetDecoder(p.Content.B)

	val.err, err = d.Error()
	assert.NoError(t, err)
	assert.ErrorIs(t, val.err, test.err)

	val.test, err = d.String()
	assert.NoError(t, err)
	assert.Equal(t, test.test, val.test)

	val.b, err = d.Bytes(nil)
	assert.NoError(t, err)
	assert.Equal(t, test.b, val.b)

	val.num1, err = d.Uint8()
	assert.NoError(t, err)
	assert.Equal(t, val.num1, val.num1)

	val.num2, err = d.Uint16()
	assert.NoError(t, err)
	assert.Equal(t, test.num2, val.num2)

	val.num3, err = d.Uint32()
	assert.NoError(t, err)
	assert.Equal(t, test.num3, val.num3)

	val.num4, err = d.Uint64()
	assert.NoError(t, err)
	assert.Equal(t, test.num4, val.num4)

	val.num5, err = d.Int32()
	assert.NoError(t, err)
	assert.Equal(t, test.num5, val.num5)

	val.num6, err = d.Int64()
	assert.NoError(t, err)
	assert.Equal(t, test.num6, val.num6)

	val.num7, err = d.Float32()
	assert.NoError(t, err)
	assert.Equal(t, test.num7, val.num7)

	val.num8, err = d.Float64()
	assert.NoError(t, err)
	assert.Equal(t, test.num8, val.num8)

	val.truth, err = d.Bool()
	assert.NoError(t, err)
	assert.Equal(t, test.truth, val.truth)

	isNil := d.Nil()
	assert.True(t, isNil)

	var size uint32
	size, err = d.Slice(StringKind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(test.slice)), size)
	val.slice = make([]string, size)
	for i := range val.slice {
		val.slice[i], err = d.String()
		assert.NoError(t, err)
		assert.Equal(t, test.slice[i], val.slice[i])
	}
	assert.Equal(t, test.slice, val.slice)

	size, err = d.Map(Uint32Kind, AnyKind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(test.m)), size)
	val.m = make(map[uint32]*embedStruct, size)
	var k uint32
	var v *embedStruct
	for i := uint32(0); i < size; i++ {
		v = new(embedStruct)
		k, err = d.Uint32()
		assert.NoError(t, err)
		v.test, err = d.String()
		assert.NoError(t, err)
		v.b, err = d.Bytes(v.b)
		assert.NoError(t, err)
		val.m[k] = v
	}
	assert.Equal(t, test.m, val.m)

	assert.Equal(t, 0, len(d.b))
	d.Return()

	p.Content.Reset()
	n := testing.AllocsPerRun(100, func() {
		Encoder(p).Error(test.err).String(test.test).Bytes(test.b).Uint8(test.num1).Uint16(test.num2).Uint32(test.num3).Uint64(test.num4).Bool(test.truth).Nil()
		d = GetDecoder(p.Content.B)
		val.err, err = d.Error()
		val.test, err = d.String()
		val.b, err = d.Bytes(val.b)
		val.num1, err = d.Uint8()
		val.num2, err = d.Uint16()
		val.num3, err = d.Uint32()
		val.num4, err = d.Uint64()
		val.truth, err = d.Bool()
		isNil = d.Nil()
		d.Return()
		p.Content.Reset()
	})
	assert.Equal(t, float64(3), n)
	Put(p)
}

func TestNilSlice(t *testing.T) {
	s := make([]string, 0)
	p := Get()
	Encoder(p).Slice(uint32(len(s)), StringKind)

	d := GetDecoder(p.Content.B)
	j, err := d.Slice(StringKind)
	assert.NoError(t, err)
	assert.Equal(t, uint32(len(s)), j)

	j, err = d.Slice(StringKind)
	assert.ErrorIs(t, err, InvalidSlice)
	assert.Zero(t, j)
}

func TestError(t *testing.T) {
	t.Parallel()

	v := errors.New("Test Error")

	p := Get()
	Encoder(p).Error(v)

	d := GetDecoder(p.Content.B)
	_, err := d.String()
	assert.ErrorIs(t, err, InvalidString)

	val, err := d.Error()
	assert.NoError(t, err)
	assert.ErrorIs(t, val, v)

	d.Return()
	Put(p)
}

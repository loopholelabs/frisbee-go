/*
	Copyright 2023 Loophole Labs

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

package polyglot

import (
	"math"
	"reflect"
	"unsafe"
)

type Kind []byte

var (
	NilKind     = Kind([]byte{0})
	SliceKind   = Kind([]byte{1})
	MapKind     = Kind([]byte{2})
	AnyKind     = Kind([]byte{3})
	BytesKind   = Kind([]byte{4})
	StringKind  = Kind([]byte{5})
	ErrorKind   = Kind([]byte{6})
	BoolKind    = Kind([]byte{7})
	Uint8Kind   = Kind([]byte{8})
	Uint16Kind  = Kind([]byte{9})
	Uint32Kind  = Kind([]byte{10})
	Uint64Kind  = Kind([]byte{11})
	Int32Kind   = Kind([]byte{12})
	Int64Kind   = Kind([]byte{13})
	Float32Kind = Kind([]byte{14})
	Float64Kind = Kind([]byte{15})
)

type Error string

func (e Error) Error() string {
	return string(e)
}

func (e Error) Is(err error) bool {
	return e.Error() == err.Error()
}

var (
	falseBool = byte(0)
	trueBool  = byte(1)
)

func encodeNil(b *Buffer) {
	b.Write(NilKind)
}

func encodeMap(b *Buffer, size uint32, keyKind, valueKind Kind) {
	b.Write(MapKind)
	b.Write(keyKind)
	b.Write(valueKind)
	encodeUint32(b, size)
}

func encodeSlice(b *Buffer, size uint32, kind Kind) {
	b.Write(SliceKind)
	b.Write(kind)
	encodeUint32(b, size)
}

func encodeBytes(b *Buffer, value []byte) {
	b.Write(BytesKind)
	encodeUint32(b, uint32(len(value)))
	b.Write(value)
}

func encodeString(b *Buffer, value string) {
	var nb []byte
	/* #nosec G103 */
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&nb))
	/* #nosec G103 */
	sh := (*reflect.StringHeader)(unsafe.Pointer(&value))
	bh.Data = sh.Data
	bh.Cap = sh.Len
	bh.Len = sh.Len
	b.Write(StringKind)
	encodeUint32(b, uint32(len(nb)))
	b.Write(nb)
}

func encodeError(b *Buffer, err error) {
	b.Write(ErrorKind)
	encodeString(b, err.Error())
}

func encodeBool(b *Buffer, value bool) {
	b.Write(BoolKind)
	if value {
		*b = append(*b, trueBool)
	} else {
		*b = append(*b, falseBool)
	}
}

func encodeUint8(b *Buffer, value uint8) {
	b.Write(Uint8Kind)
	*b = append(*b, value)
}

// Variable integer encoding with the same format as binary.varint
// (https://developers.google.com/protocol-buffers/docs/encoding#varints)
func encodeUint16(b *Buffer, value uint16) {
	b.Write(Uint16Kind)
	for value >= continuation {
		// Append the lower 7 bits of the value, then shift the value to the right by 7 bits.
		*b = append(*b, byte(value)|continuation)
		value >>= 7
	}
	*b = append(*b, byte(value))
}

func encodeUint32(b *Buffer, value uint32) {
	b.Write(Uint32Kind)
	for value >= continuation {
		// Append the lower 7 bits of the value, then shift the value to the right by 7 bits.
		*b = append(*b, byte(value)|continuation)
		value >>= 7
	}
	*b = append(*b, byte(value))
}

func encodeUint64(b *Buffer, value uint64) {
	b.Write(Uint64Kind)
	for value >= continuation {
		// Append the lower 7 bits of the value, then shift the value to the right by 7 bits.
		*b = append(*b, byte(value)|continuation)
		value >>= 7
	}
	*b = append(*b, byte(value))
}

func encodeInt32(b *Buffer, value int32) {
	b.Write(Int32Kind)
	// Shift the value to the left by 1 bit, then flip the bits if the value is negative.
	castValue := uint32(value) << 1
	if value < 0 {
		castValue = ^castValue
	}
	for castValue >= continuation {
		// Append the lower 7 bits of the value, then shift the value to the right by 7 bits.
		*b = append(*b, byte(castValue)|continuation)
		castValue >>= 7
	}
	*b = append(*b, byte(castValue))
}

func encodeInt64(b *Buffer, value int64) {
	b.Write(Int64Kind)
	// Shift the value to the left by 1 bit, then flip the bits if the value is negative.
	castValue := uint64(value) << 1
	if value < 0 {
		castValue = ^castValue
	}
	for castValue >= continuation {
		// Append the lower 7 bits of the value, then shift the value to the right by 7 bits.
		*b = append(*b, byte(castValue)|continuation)
		castValue >>= 7
	}
	*b = append(*b, byte(castValue))
}

func encodeFloat32(b *Buffer, value float32) {
	b.Write(Float32Kind)
	castValue := math.Float32bits(value)
	*b = append(*b, byte(castValue>>24), byte(castValue>>16), byte(castValue>>8), byte(castValue))
}

func encodeFloat64(b *Buffer, value float64) {
	b.Write(Float64Kind)
	castValue := math.Float64bits(value)
	*b = append(*b, byte(castValue>>56), byte(castValue>>48), byte(castValue>>40), byte(castValue>>32), byte(castValue>>24), byte(castValue>>16), byte(castValue>>8), byte(castValue))
}

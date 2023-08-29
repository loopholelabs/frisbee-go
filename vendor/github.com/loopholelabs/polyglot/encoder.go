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

type BufferEncoder Buffer

func Encoder(b *Buffer) *BufferEncoder {
	return (*BufferEncoder)(b)
}

func (e *BufferEncoder) Nil() *BufferEncoder {
	encodeNil((*Buffer)(e))
	return e
}

func (e *BufferEncoder) Map(size uint32, keyKind, valueKind Kind) *BufferEncoder {
	encodeMap((*Buffer)(e), size, keyKind, valueKind)
	return e
}

func (e *BufferEncoder) Slice(size uint32, kind Kind) *BufferEncoder {
	encodeSlice((*Buffer)(e), size, kind)
	return e
}

func (e *BufferEncoder) Bytes(value []byte) *BufferEncoder {
	encodeBytes((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) String(value string) *BufferEncoder {
	encodeString((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) Error(value error) *BufferEncoder {
	encodeError((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) Bool(value bool) *BufferEncoder {
	encodeBool((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) Uint8(value uint8) *BufferEncoder {
	encodeUint8((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) Uint16(value uint16) *BufferEncoder {
	encodeUint16((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) Uint32(value uint32) *BufferEncoder {
	encodeUint32((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) Uint64(value uint64) *BufferEncoder {
	encodeUint64((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) Int32(value int32) *BufferEncoder {
	encodeInt32((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) Int64(value int64) *BufferEncoder {
	encodeInt64((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) Float32(value float32) *BufferEncoder {
	encodeFloat32((*Buffer)(e), value)
	return e
}

func (e *BufferEncoder) Float64(value float64) *BufferEncoder {
	encodeFloat64((*Buffer)(e), value)
	return e
}

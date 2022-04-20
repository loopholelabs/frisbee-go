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
	"sync"
)

var decoderPool sync.Pool

type Decoder struct {
	b []byte
}

func GetDecoder(b []byte) (d *Decoder) {
	v := decoderPool.Get()
	if v == nil {
		d = &Decoder{
			b: b,
		}
	} else {
		d = v.(*Decoder)
		d.b = b
	}
	return
}

func ReturnDecoder(d *Decoder) {
	if d != nil {
		d.b = nil
		decoderPool.Put(d)
	}
}

func (d *Decoder) Return() {
	ReturnDecoder(d)
}

func (d *Decoder) Nil() (value bool) {
	d.b, value = decodeNil(d.b)
	return
}

func (d *Decoder) Map(keyKind Kind, valueKind Kind) (size uint32, err error) {
	d.b, size, err = decodeMap(d.b, keyKind, valueKind)
	return
}

func (d *Decoder) Slice(kind Kind) (size uint32, err error) {
	d.b, size, err = decodeSlice(d.b, kind)
	return
}

func (d *Decoder) Bytes(b []byte) (value []byte, err error) {
	d.b, value, err = decodeBytes(d.b, b)
	return
}

func (d *Decoder) String() (value string, err error) {
	d.b, value, err = decodeString(d.b)
	return
}

func (d *Decoder) Error() (value error, err error) {
	d.b, value, err = decodeError(d.b)
	return
}

func (d *Decoder) Bool() (value bool, err error) {
	d.b, value, err = decodeBool(d.b)
	return
}

func (d *Decoder) Uint8() (value uint8, err error) {
	d.b, value, err = decodeUint8(d.b)
	return
}

func (d *Decoder) Uint16() (value uint16, err error) {
	d.b, value, err = decodeUint16(d.b)
	return
}

func (d *Decoder) Uint32() (value uint32, err error) {
	d.b, value, err = decodeUint32(d.b)
	return
}

func (d *Decoder) Uint64() (value uint64, err error) {
	d.b, value, err = decodeUint64(d.b)
	return
}

func (d *Decoder) Int32() (value int32, err error) {
	d.b, value, err = decodeInt32(d.b)
	return
}

func (d *Decoder) Int64() (value int64, err error) {
	d.b, value, err = decodeInt64(d.b)
	return
}

func (d *Decoder) Float32() (value float32, err error) {
	d.b, value, err = decodeFloat32(d.b)
	return
}

func (d *Decoder) Float64() (value float64, err error) {
	d.b, value, err = decodeFloat64(d.b)
	return
}

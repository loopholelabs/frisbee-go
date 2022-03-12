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
	"math"
	"reflect"
	"unsafe"
)

var (
	falseBool = byte(0)
	trueBool  = byte(1)
)

func encodeNil(p *Packet) {
	p.Content.Write(NilKind)
}

func encodeMap(p *Packet, size uint32, keyKind Kind, valueKind Kind) {
	p.Content.Write(MapKind)
	p.Content.Write(keyKind)
	p.Content.Write(valueKind)
	encodeUint32(p, size)
}

func encodeSlice(p *Packet, size uint32, kind Kind) {
	p.Content.Write(SliceKind)
	p.Content.Write(kind)
	encodeUint32(p, size)
}

func encodeBytes(p *Packet, value []byte) {
	p.Content.Write(BytesKind)
	encodeUint32(p, uint32(len(value)))
	p.Content.Write(value)
}

func encodeString(p *Packet, value string) {
	var b []byte
	/* #nosec G103 */
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	/* #nosec G103 */
	sh := (*reflect.StringHeader)(unsafe.Pointer(&value))
	bh.Data = sh.Data
	bh.Cap = sh.Len
	bh.Len = sh.Len
	p.Content.Write(StringKind)
	encodeUint32(p, uint32(len(b)))
	p.Content.Write(b)
}

func encodeError(p *Packet, err error) {
	p.Content.Write(ErrorKind)
	encodeString(p, err.Error())
}

func encodeBool(p *Packet, value bool) {
	p.Content.Write(BoolKind)
	if value {
		p.Content.B = append(p.Content.B, trueBool)
	} else {
		p.Content.B = append(p.Content.B, falseBool)
	}
}

func encodeUint8(p *Packet, value uint8) {
	p.Content.Write(Uint8Kind)
	p.Content.B = append(p.Content.B, value)
}

func encodeUint16(p *Packet, value uint16) {
	p.Content.Write(Uint16Kind)
	p.Content.B = append(p.Content.B, byte(value>>8), byte(value))
}

func encodeUint32(p *Packet, value uint32) {
	p.Content.Write(Uint32Kind)
	p.Content.B = append(p.Content.B, byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
}

func encodeUint64(p *Packet, value uint64) {
	p.Content.Write(Uint64Kind)
	p.Content.B = append(p.Content.B, byte(value>>56), byte(value>>48), byte(value>>40), byte(value>>32), byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
}

func encodeInt32(p *Packet, value int32) {
	p.Content.Write(Int32Kind)
	castValue := uint32(value)
	p.Content.B = append(p.Content.B, byte(castValue>>24), byte(castValue>>16), byte(castValue>>8), byte(castValue))
}

func encodeInt64(p *Packet, value int64) {
	p.Content.Write(Int64Kind)
	castValue := uint64(value)
	p.Content.B = append(p.Content.B, byte(castValue>>56), byte(castValue>>48), byte(castValue>>40), byte(castValue>>32), byte(castValue>>24), byte(castValue>>16), byte(castValue>>8), byte(castValue))
}

func encodeFloat32(p *Packet, value float32) {
	p.Content.Write(Float32Kind)
	castValue := math.Float32bits(value)
	p.Content.B = append(p.Content.B, byte(castValue>>24), byte(castValue>>16), byte(castValue>>8), byte(castValue))
}

func encodeFloat64(p *Packet, value float64) {
	p.Content.Write(Float64Kind)
	castValue := math.Float64bits(value)
	p.Content.B = append(p.Content.B, byte(castValue>>56), byte(castValue>>48), byte(castValue>>40), byte(castValue>>32), byte(castValue>>24), byte(castValue>>16), byte(castValue>>8), byte(castValue))
}

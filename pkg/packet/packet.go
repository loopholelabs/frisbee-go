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
	"github.com/loopholelabs/frisbee/pkg/content"
	"github.com/loopholelabs/frisbee/pkg/metadata"
)

// Packet is the structured frisbee data packet, and contains the following:
//
//	type Packet struct {
//		Metadata struct {
//			Id            uint16 // 2 Bytes
//			Operation     uint16 // 2 Bytes
//			ContentLength uint32 // 4 Bytes
//		}
//		Content *content.Content
//	}
//
// The ID field can be used however the user sees fit, however ContentLength must match the length of the content being
// delivered with the frisbee packet (see the Async.WritePacket function for more details), and the Operation field must be greater than uint16(9).
type Packet struct {
	Metadata *metadata.Metadata
	Content  *content.Content
}

func (p *Packet) Reset() {
	p.Metadata.Id = 0
	p.Metadata.Operation = 0
	p.Metadata.ContentLength = 0
	p.Content.Reset()
}

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
	Int32Kind   = Kind([]byte{11})
	Int64Kind   = Kind([]byte{12})
	Float32Kind = Kind([]byte{13})
	Float64Kind = Kind([]byte{14})
)

type Error string

func (e Error) Error() string {
	return string(e)
}

func (e Error) Is(err error) bool {
	return e.Error() == err.Error()
}

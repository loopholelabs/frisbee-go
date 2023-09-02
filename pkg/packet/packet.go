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
	"encoding/binary"
	"github.com/loopholelabs/frisbee-go/pkg/metadata"
	"github.com/loopholelabs/polyglot"
	"io"
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
	Content  *polyglot.Buffer
}

func (p *Packet) Reset() {
	p.Metadata.Id = 0
	p.Metadata.Operation = 0
	p.Metadata.ContentLength = 0
	p.Content.Reset()
}

func (p *Packet) Write(w io.Writer) error {
	encodedMetadata := metadata.GetBuffer()
	binary.BigEndian.PutUint16(encodedMetadata[metadata.IdOffset:metadata.IdOffset+metadata.IdSize], p.Metadata.Id)
	binary.BigEndian.PutUint16(encodedMetadata[metadata.OperationOffset:metadata.OperationOffset+metadata.OperationSize], p.Metadata.Operation)
	binary.BigEndian.PutUint32(encodedMetadata[metadata.ContentLengthOffset:metadata.ContentLengthOffset+metadata.ContentLengthSize], p.Metadata.ContentLength)

	_, err := w.Write(encodedMetadata[:])
	metadata.PutBuffer(encodedMetadata)

	if err != nil {
		return err
	}

	if p.Metadata.ContentLength > 0 {
		_, err = w.Write((*p.Content)[:p.Metadata.ContentLength])
		if err != nil {
			return err
		}
	}

	return nil
}

func Read(r io.Reader) (*Packet, error) {
	var encodedPacket metadata.Buffer

	_, err := io.ReadAtLeast(r, encodedPacket[:], metadata.Size)
	if err != nil {
		return nil, err
	}

	p := Get()

	p.Metadata.Id = binary.BigEndian.Uint16(encodedPacket[metadata.IdOffset : metadata.IdOffset+metadata.IdSize])
	p.Metadata.Operation = binary.BigEndian.Uint16(encodedPacket[metadata.OperationOffset : metadata.OperationOffset+metadata.OperationSize])
	p.Metadata.ContentLength = binary.BigEndian.Uint32(encodedPacket[metadata.ContentLengthOffset : metadata.ContentLengthOffset+metadata.ContentLengthSize])

	if p.Metadata.ContentLength > 0 {
		for cap(*p.Content) < int(p.Metadata.ContentLength) {
			*p.Content = append((*p.Content)[:cap(*p.Content)], 0)
		}
		*p.Content = (*p.Content)[:p.Metadata.ContentLength]
		_, err = io.ReadAtLeast(r, *p.Content, int(p.Metadata.ContentLength))
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

func New() *Packet {
	return &Packet{
		Metadata: new(metadata.Metadata),
		Content:  polyglot.NewBuffer(),
	}
}

// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"github.com/loopholelabs/polyglot/v2"

	"github.com/loopholelabs/frisbee-go/pkg/metadata"
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

func New() *Packet {
	return &Packet{
		Metadata: new(metadata.Metadata),
		Content:  polyglot.NewBuffer(),
	}
}

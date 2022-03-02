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
//		Content []byte
//	}
//
// The ID field can be used however the user sees fit, however ContentLength must match the length of the content being
// delivered with the frisbee packet (see the Async.WritePacket function for more details), and the Operation field must be greater than uint16(9).
type Packet struct {
	Metadata *metadata.Metadata
	Content  *Content
}

func (p *Packet) Reset() {
	p.Metadata.Id = 0
	p.Metadata.Operation = 0
	p.Metadata.ContentLength = 0
	p.Content.Reset()
}

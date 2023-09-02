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

package frisbee

import (
	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"sync"
)

type Packets struct {
	mu sync.Mutex
	d  []*packet.Packet
}

// NewPackets do create wrapper around list of packets allowing to operate on
// multithreading environment
func NewPackets() *Packets {
	return &Packets{}
}

// Poll get and retrieve a packed out of the list
func (s *Packets) Poll() *packet.Packet {
	s.mu.Lock()
	if len(s.d) > 0 {
		var p *packet.Packet
		p, s.d = s.d[0], s.d[1:]
		s.mu.Unlock()
		return p
	}
	s.mu.Unlock()
	return nil
}

// Set setting the full packet list to point to a new list
func (s *Packets) Set(packets []*packet.Packet) {
	s.mu.Lock()
	if len(packets) > 0 {
		s.d = make([]*packet.Packet, len(packets))
		copy(s.d, packets)
	} else {
		s.d = nil
	}
	s.mu.Unlock()
}

// Append into existing packet list new packet elements
func (s *Packets) Append(packets []*packet.Packet) {
	s.mu.Lock()
	if len(packets) > 0 {
		if len(s.d) == 0 {
			s.d = make([]*packet.Packet, len(packets))
			copy(s.d, packets)
		} else {
			for _, p := range packets {
				s.d = append(s.d, p)
			}
		}
	}
	s.mu.Unlock()
}

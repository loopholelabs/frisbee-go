package frisbee

import (
	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"sync"
)

type Packets struct {
	mu sync.Mutex
	d  []*packet.Packet
}

func NewPackets() *Packets {
	return &Packets{}
}

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

func (s *Packets) Append(packets []*packet.Packet) {
	s.mu.Lock()
	if len(packets) > 0 {
		if len(s.d) <= 0 {
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

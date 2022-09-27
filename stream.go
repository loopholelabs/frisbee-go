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

package frisbee

import (
	"github.com/loopholelabs/common/pkg/queue"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"go.uber.org/atomic"
	"sync"
)

type Stream struct {
	id      uint16
	conn    *Async
	closed  *atomic.Bool
	queue   *queue.Circular[packet.Packet, *packet.Packet]
	staleMu sync.Mutex
	stale   []*packet.Packet
}

func NewStream(id uint16, conn *Async) *Stream {
	return &Stream{
		id:     id,
		conn:   conn,
		closed: atomic.NewBool(false),
		queue:  queue.NewCircular[packet.Packet, *packet.Packet](DefaultStreamBufferSize),
	}
}

func (s *Stream) push(p *packet.Packet) error {
	return s.queue.Push(p)
}

// ReadPacket is a blocking function that will wait until a Frisbee packet is available and then return it (and its content).
// In the event that the connection is closed, ReadPacket will return an error.
func (s *Stream) ReadPacket() (*packet.Packet, error) {
	if s.closed.Load() {
		s.staleMu.Lock()
		if len(s.stale) > 0 {
			var p *packet.Packet
			p, s.stale = s.stale[0], s.stale[1:]
			s.staleMu.Unlock()
			return p, nil
		}
		s.staleMu.Unlock()
		return nil, ConnectionClosed
	}

	readPacket, err := s.queue.Pop()
	if err != nil {
		if s.closed.Load() {
			s.staleMu.Lock()
			if len(s.stale) > 0 {
				var p *packet.Packet
				p, s.stale = s.stale[0], s.stale[1:]
				s.staleMu.Unlock()
				return p, nil
			}
			s.staleMu.Unlock()
			return nil, ConnectionClosed
		}
		return nil, err
	}

	return readPacket, nil
}

//func (s *Stream) Close() {
//	s.staleMu.Lock()
//	if s.closed.CAS(false, true) {
//		s.Logger().Debug().Msg("connection close called, killing goroutines")
//		s.killGoroutines()
//		s.staleMu.Unlock()
//		s.Lock()
//		if s.writer.Buffered() > 0 {
//			_ = s.conn.Flush()
//			_ = s.conn.SetWriteDeadline(emptyTime)
//		}
//		s.Unlock()
//		return nil
//	}
//	s.staleMu.Unlock()
//	return ConnectionClosed
//}

// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"sync"
	"sync/atomic"

	"github.com/loopholelabs/common/pkg/queue"

	"github.com/loopholelabs/frisbee-go/pkg/packet"
)

// DefaultStreamBufferSize is the default size of the stream buffer.
const DefaultStreamBufferSize = 1 << 12

type NewStreamHandler func(*Stream)

type Stream struct {
	id      uint16
	conn    *Async
	closed  atomic.Bool
	queue   *queue.Circular[packet.Packet, *packet.Packet]
	staleMu sync.Mutex
	stale   []*packet.Packet
}

func newStream(id uint16, conn *Async) *Stream {
	return &Stream{
		id:    id,
		conn:  conn,
		queue: queue.NewCircular[packet.Packet, *packet.Packet](DefaultStreamBufferSize),
	}
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
		return nil, StreamClosed
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
			return nil, StreamClosed
		}
		return nil, err
	}

	return readPacket, nil
}

// WritePacket will write the given packet to the stream but the ID and Operation will be
// overwritten with the stream's ID and the STREAM operation. Packets send to a stream
// must have a ContentLength greater than 0.
func (s *Stream) WritePacket(p *packet.Packet) error {
	if s.closed.Load() {
		return StreamClosed
	}
	if p.Metadata.ContentLength == 0 {
		return InvalidStreamPacket
	}
	p.Metadata.Id = s.id
	p.Metadata.Operation = STREAM
	return s.conn.writePacket(p, true)
}

// ID returns the stream's ID.
func (s *Stream) ID() uint16 {
	return s.id
}

// Conn returns the connection that the stream is associated with.
func (s *Stream) Conn() *Async {
	return s.conn
}

// Close will close the stream and prevent any further reads or writes.
func (s *Stream) Close() error {
	return s.closeSend(true)
}

func (s *Stream) closeSend(lock bool) error {
	s.staleMu.Lock()
	if s.closed.CompareAndSwap(false, true) {
		s.queue.Close()
		s.stale = s.queue.Drain()
		s.staleMu.Unlock()

		p := packet.Get()
		p.Metadata.Id = s.id
		p.Metadata.Operation = STREAM
		err := s.conn.writePacket(p, true)
		packet.Put(p)

		if lock {
			s.conn.streamsMu.Lock()
			delete(s.conn.streams, s.id)
			s.conn.streamsMu.Unlock()
		}

		return err
	}
	s.staleMu.Unlock()
	return StreamClosed
}

// close will close the stream and prevent any further reads or writes without sending a stream close packet.
func (s *Stream) close() {
	s.staleMu.Lock()
	if s.closed.CompareAndSwap(false, true) {
		s.queue.Close()
		s.stale = s.queue.Drain()
	}
	s.staleMu.Unlock()
}

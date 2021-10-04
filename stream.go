/*
	Copyright 2021 Loophole Labs

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
	"encoding/binary"
	"github.com/loopholelabs/frisbee/internal/errors"
	"github.com/loopholelabs/frisbee/internal/protocol"
	"go.uber.org/atomic"
	"io"
)

type Stream struct {
	*Async
	id             uint64
	incomingBuffer *incomingBuffer
	closed         *atomic.Bool
}

func (c *Async) NewStream(id uint64) *Stream {
	streamConn := &Stream{
		Async:          c,
		id:             id,
		incomingBuffer: newIncomingBuffer(),
		closed:         atomic.NewBool(false),
	}

	c.streamConnMutex.Lock()
	c.streamConns[id] = streamConn
	c.streamConnMutex.Unlock()

	return streamConn
}

func (s *Stream) ID() uint64 {
	return s.id
}

func (s *Stream) Closed() bool {
	return s.closed.Load()
}

func (s *Stream) Close() error {
	s.closed.Store(true)
	return s.WriteMessage(&Message{
		Id:            s.id,
		Operation:     STREAMCLOSE,
		ContentLength: 0,
	}, nil)
}

// Write takes a byte slice and sends a NEWSTREAM message
func (s *Stream) Write(p []byte) (int, error) {
	encodedMessage := [protocol.MessageSize]byte{protocol.ReservedBytes[0], protocol.ReservedBytes[1], protocol.ReservedBytes[2], protocol.ReservedBytes[3]}

	binary.BigEndian.PutUint64(encodedMessage[protocol.IdOffset:protocol.IdOffset+protocol.IdSize], s.id)
	binary.BigEndian.PutUint32(encodedMessage[protocol.OperationOffset:protocol.OperationOffset+protocol.OperationSize], NEWSTREAM)
	binary.BigEndian.PutUint64(encodedMessage[protocol.ContentLengthOffset:protocol.ContentLengthOffset+protocol.ContentLengthSize], uint64(len(p)))

	s.Lock()
	if s.state.Load() != CONNECTED {
		s.Unlock()
		return 0, s.Error()
	}

	if s.Closed() {
		s.Unlock()
		return 0, ConnectionClosed
	}

	_, err := s.writer.Write(encodedMessage[:])
	if err != nil {
		s.Unlock()
		if s.state.Load() != CONNECTED {
			err = s.Error()
			s.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
			return 0, errors.WithContext(err, WRITE)
		}
		s.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
		return 0, s.closeWithError(err)
	}

	_, err = s.writer.Write(p)
	if err != nil {
		s.Unlock()
		if s.state.Load() != CONNECTED {
			err = s.Error()
			s.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
			return 0, errors.WithContext(err, WRITE)
		}
		s.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
		return 0, s.closeWithError(err)
	}

	if len(s.flusher) == 0 {
		select {
		case s.flusher <- struct{}{}:
		default:
		}
	}

	s.Unlock()
	return len(p), nil
}

// ReadFrom is a function that will send NEWSTREAM messages from an io.Reader until EOF or an error occurs
// In the event that the connection is closed, ReadFrom will return an error.
func (s *Stream) ReadFrom(r io.Reader) (n int64, err error) {
	buf := make([]byte, DefaultBufferSize)

	encodedMessage := [protocol.MessageSize]byte{protocol.ReservedBytes[0], protocol.ReservedBytes[1], protocol.ReservedBytes[2], protocol.ReservedBytes[3]}
	binary.BigEndian.PutUint64(encodedMessage[protocol.IdOffset:protocol.IdOffset+protocol.IdSize], s.id)
	binary.BigEndian.PutUint32(encodedMessage[protocol.OperationOffset:protocol.OperationOffset+protocol.OperationSize], NEWSTREAM)

	for {
		var nn int
		if s.state.Load() != CONNECTED {
			return n, err
		}

		if s.Closed() {
			return n, ConnectionClosed
		}
		nn, err = r.Read(buf)
		if nn == 0 || err != nil {
			break
		}

		n += int64(nn)

		binary.BigEndian.PutUint64(encodedMessage[protocol.ContentLengthOffset:protocol.ContentLengthOffset+protocol.ContentLengthSize], uint64(nn))

		s.Lock()

		_, err := s.writer.Write(encodedMessage[:])
		if err != nil {
			s.Unlock()
			if s.state.Load() != CONNECTED {
				err = s.Error()
				s.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
				return n, errors.WithContext(err, WRITE)
			}
			s.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
			return n, s.closeWithError(err)
		}

		_, err = s.writer.Write(buf[:nn])
		if err != nil {
			s.Unlock()
			if s.state.Load() != CONNECTED {
				err = s.Error()
				s.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
				return n, errors.WithContext(err, WRITE)
			}
			s.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
			return n, s.closeWithError(err)
		}

		if len(s.flusher) == 0 {
			select {
			case s.flusher <- struct{}{}:
			default:
			}
		}
		s.Unlock()
	}

	if errors.Is(err, io.EOF) {
		err = nil
	}

	return
}

// Read is a function that will read buffer messages into a byte slice.
// In the event that the connection is closed, Read will return an error.
func (s *Stream) Read(p []byte) (int, error) {
LOOP:
	s.incomingBuffer.Lock()
	if s.state.Load() != CONNECTED {
		s.incomingBuffer.Unlock()
		return 0, ConnectionClosed
	}
	if s.Closed() {
		s.incomingBuffer.Unlock()
		return 0, ConnectionClosed
	}
	for s.incomingBuffer.buffer.Len() == 0 {
		s.incomingBuffer.Unlock()
		goto LOOP
	}
	defer s.incomingBuffer.Unlock()
	return s.incomingBuffer.buffer.Read(p)
}

// WriteTo is a function that will read buffer messages into an io.Writer until EOF or an error occurs
// In the event that the connection is closed, WriteTo will return an error.
func (s *Stream) WriteTo(w io.Writer) (n int64, err error) {
	for err == nil {
		s.incomingBuffer.Lock()
		var nn int
		if s.state.Load() != CONNECTED {
			s.incomingBuffer.Unlock()
			return n, ConnectionClosed
		}
		if s.Closed() {
			s.incomingBuffer.Unlock()
			return n, ConnectionClosed
		}
		if s.incomingBuffer.buffer.Len() == 0 {
			s.incomingBuffer.Unlock()
			continue
		} else {
			nn, err = w.Write(s.incomingBuffer.buffer.Bytes())
			if nn > 0 {
				s.incomingBuffer.buffer.Next(nn)
				n += int64(nn)
			}
		}
		s.incomingBuffer.Unlock()
	}

	if errors.Is(err, io.EOF) {
		err = nil
	}
	return
}

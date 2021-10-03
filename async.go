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
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"github.com/loopholelabs/frisbee/internal/errors"
	"github.com/loopholelabs/frisbee/internal/protocol"
	"github.com/loopholelabs/frisbee/internal/ringbuffer"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

// Async is the underlying asynchronous frisbee connection which has extremely efficient read and write logic and
// can handle the specific frisbee requirements. This is not meant to be used on its own, and instead is
// meant to be used by frisbee client and server implementations
type Async struct {
	sync.Mutex
	conn             net.Conn
	state            *atomic.Int32
	writer           *bufio.Writer
	flusher          chan struct{}
	incomingMessages *ringbuffer.RingBuffer
	streamConnMutex  sync.RWMutex
	streamConns      map[uint64]*Stream
	streamConnCh     chan *Stream
	logger           *zerolog.Logger
	wg               sync.WaitGroup
	error            *atomic.Error
}

// ConnectAsync creates a new TCP connection (using net.Dial) and wraps it in a frisbee connection
func ConnectAsync(network string, addr string, keepAlive time.Duration, logger *zerolog.Logger, TLSConfig *tls.Config) (*Async, error) {
	var conn net.Conn
	var err error

	if TLSConfig != nil {
		conn, err = tls.Dial(network, addr, TLSConfig)
	} else {
		conn, err = net.Dial(network, addr)
	}

	if err != nil {
		return nil, errors.WithContext(err, DIAL)
	}
	_ = conn.(*net.TCPConn).SetKeepAlive(true)
	_ = conn.(*net.TCPConn).SetKeepAlivePeriod(keepAlive)

	return NewAsync(conn, logger), nil
}

// NewAsync takes an existing net.Conn object and wraps it in a frisbee connection
func NewAsync(c net.Conn, logger *zerolog.Logger) (conn *Async) {
	conn = &Async{
		conn:             c,
		state:            atomic.NewInt32(CONNECTED),
		writer:           bufio.NewWriterSize(c, DefaultBufferSize),
		incomingMessages: ringbuffer.NewRingBuffer(DefaultBufferSize),
		streamConns:      make(map[uint64]*Stream),
		streamConnCh:     make(chan *Stream, 1024),
		flusher:          make(chan struct{}, 1024),
		logger:           logger,
		error:            atomic.NewError(ConnectionClosed),
	}

	if logger == nil {
		conn.logger = &defaultLogger
	}

	conn.wg.Add(2)
	go conn.flushLoop()
	go conn.readLoop()

	return
}

// SetDeadline sets the read and write deadline on the underlying net.Conn
func (c *Async) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying net.Conn
func (c *Async) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying net.Conn
func (c *Async) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// ConnectionState returns the tls.ConnectionState of a *tls.Conn
// if the connection is not *tls.Conn then the NotTLSConnectionError is returned
func (c *Async) ConnectionState() (tls.ConnectionState, error) {
	if tlsConn, ok := c.conn.(*tls.Conn); ok {
		return tlsConn.ConnectionState(), nil
	}
	return tls.ConnectionState{}, NotTLSConnectionError
}

// LocalAddr returns the local address of the underlying net.Conn
func (c *Async) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote address of the underlying net.Conn
func (c *Async) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// StreamChannel returns a channel that can be listened to for new stream connections
func (c *Async) StreamChannel() <-chan *Stream {
	return c.streamConnCh
}

// WriteMessage takes a frisbee.Message and some (optional) accompanying content, and queues it up to send asynchronously.
//
// If message.ContentLength == 0, then the content array must be nil. Otherwise, it is required that message.ContentLength == len(content).
func (c *Async) WriteMessage(message *Message, content *[]byte) error {
	if content != nil && int(message.ContentLength) != len(*content) {
		return InvalidContentLength
	}

	encodedMessage := [protocol.MessageSize]byte{protocol.ReservedBytes[0], protocol.ReservedBytes[1], protocol.ReservedBytes[2], protocol.ReservedBytes[3]}

	binary.BigEndian.PutUint32(encodedMessage[protocol.FromOffset:protocol.FromOffset+protocol.FromSize], message.From)
	binary.BigEndian.PutUint32(encodedMessage[protocol.ToOffset:protocol.ToOffset+protocol.ToSize], message.To)
	binary.BigEndian.PutUint64(encodedMessage[protocol.IdOffset:protocol.IdOffset+protocol.IdSize], message.Id)
	binary.BigEndian.PutUint32(encodedMessage[protocol.OperationOffset:protocol.OperationOffset+protocol.OperationSize], message.Operation)
	binary.BigEndian.PutUint64(encodedMessage[protocol.ContentLengthOffset:protocol.ContentLengthOffset+protocol.ContentLengthSize], message.ContentLength)

	c.Lock()
	if c.state.Load() != CONNECTED {
		c.Unlock()
		return c.Error()
	}

	_, err := c.writer.Write(encodedMessage[:])
	if err != nil {
		c.Unlock()
		if c.state.Load() != CONNECTED {
			err = c.Error()
			c.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
			return errors.WithContext(err, WRITE)
		}
		c.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
		return c.closeWithError(err)
	}
	if content != nil {
		_, err = c.writer.Write(*content)
		if err != nil {
			c.Unlock()
			if c.state.Load() != CONNECTED {
				err = c.Error()
				c.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
				return errors.WithContext(err, WRITE)
			}
			c.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
			return c.closeWithError(err)
		}
	}

	if len(c.flusher) == 0 {
		select {
		case c.flusher <- struct{}{}:
		default:
		}
	}

	c.Unlock()

	return nil
}

// ReadMessage is a blocking function that will wait until a frisbee message is available and then return it (and its content).
// In the event that the connection is closed, ReadMessage will return an error.
func (c *Async) ReadMessage() (*Message, *[]byte, error) {
	if c.state.Load() != CONNECTED {
		return nil, nil, c.Error()
	}

	readPacket, err := c.incomingMessages.Pop()
	if err != nil {
		if c.state.Load() != CONNECTED {
			err = c.Error()
			c.logger.Error().Msgf(errors.WithContext(err, POP).Error())
			return nil, nil, errors.WithContext(err, POP)
		}
		c.logger.Error().Msgf(errors.WithContext(err, POP).Error())
		return nil, nil, errors.WithContext(c.closeWithError(err), POP)
	}

	return (*Message)(readPacket.Message), readPacket.Content, nil
}

// Flush allows for synchronous messaging by flushing the message buffer and instantly sending messages
func (c *Async) Flush() error {
	c.Lock()
	if c.writer.Buffered() > 0 {
		err := c.SetWriteDeadline(time.Now().Add(defaultDeadline))
		if err != nil {
			c.Unlock()
			return c.closeWithError(err)
		}
		err = c.writer.Flush()
		_ = c.SetWriteDeadline(time.Time{})
		if err != nil {
			c.Unlock()
			return c.closeWithError(err)
		}
	}
	c.Unlock()
	return nil
}

// WriteBufferSize returns the size of the underlying message buffer (used for internal message handling and for heartbeat logic)
func (c *Async) WriteBufferSize() int {
	c.Lock()
	if c.state.Load() != CONNECTED {
		c.Unlock()
		return 0
	}
	i := c.writer.Buffered()
	c.Unlock()
	return i
}

// Logger returns the underlying logger of the frisbee connection
func (c *Async) Logger() *zerolog.Logger {
	return c.logger
}

// Error returns the error that caused the frisbee.Async to close or go into a paused state
func (c *Async) Error() error {
	return c.error.Load()
}

// Raw shuts off all of frisbee's underlying functionality and converts the frisbee connection into a normal TCP connection (net.Conn)
func (c *Async) Raw() net.Conn {
	_ = c.close()
	return c.conn
}

// Close closes the frisbee connection gracefully
func (c *Async) Close() error {
	err := c.close()
	if errors.Is(err, ConnectionClosed) {
		return nil
	}
	_ = c.conn.Close()
	return err
}

func (c *Async) killGoroutines() {
	c.Lock()
	c.incomingMessages.Close()
	close(c.flusher)
	c.Unlock()
	_ = c.SetDeadline(time.Now())
	c.wg.Wait()
	_ = c.SetDeadline(time.Time{})
}

func (c *Async) pause() error {
	if c.state.CAS(CONNECTED, PAUSED) {
		c.error.Store(ConnectionPaused)
		c.killGoroutines()
		return nil
	} else if c.state.Load() == PAUSED {
		return ConnectionPaused
	}
	return ConnectionClosed
}

func (c *Async) close() error {
	if c.state.CAS(CONNECTED, CLOSED) {
		c.error.Store(ConnectionClosed)
		c.killGoroutines()
		c.Lock()
		if c.writer.Buffered() > 0 {
			_ = c.writer.Flush()
		}
		c.Unlock()
		return nil
	} else if c.state.CAS(PAUSED, CLOSED) {
		c.error.Store(ConnectionClosed)
		return nil
	}
	return ConnectionClosed
}

func (c *Async) closeWithError(err error) error {
	if os.IsTimeout(err) {
		return err
	} else if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
		pauseError := c.pause()
		if errors.Is(pauseError, ConnectionClosed) {
			c.Logger().Debug().Msgf("attempted to close connection with error, but connection already closed (inner error: %+v)", err)
			return ConnectionClosed
		} else {
			c.Logger().Debug().Msgf("attempted to close connection with error, but error was EOF so pausing connection instead (inner error: %+v)", err)
			return ConnectionPaused
		}
	} else {
		closeError := c.close()
		if errors.Is(closeError, ConnectionClosed) {
			c.Logger().Debug().Msgf("attempted to close connection with error, but connection already closed (inner error: %+v)", err)
			return ConnectionClosed
		} else {
			c.Logger().Debug().Msgf("closing connection with error: %+v", err)
		}
	}
	c.error.Store(err)
	_ = c.conn.Close()
	return err
}

func (c *Async) flushLoop() {
	defer c.wg.Done()
	for {
		if _, ok := <-c.flusher; !ok {
			return
		}
		err := c.Flush()
		if err != nil {
			_ = c.closeWithError(err)
			return
		}
	}
}

func (c *Async) readLoop() {
	defer c.wg.Done()
	buf := make([]byte, DefaultBufferSize)
	var index int
	for {
		buf = buf[:cap(buf)]
		if len(buf) < protocol.MessageSize {
			_ = c.closeWithError(InvalidBufferLength)
			return
		}

		var n int
		var err error
		err = c.SetReadDeadline(time.Now().Add(defaultDeadline))
		if err != nil {
			_ = c.closeWithError(err)
			return
		}
		for n < protocol.MessageSize {
			var nn int
			nn, err = c.conn.Read(buf[n:])
			if n == 0 {
				_ = c.SetReadDeadline(time.Time{})
			}
			n += nn
			if err != nil {
				if n < protocol.MessageSize {
					_ = c.closeWithError(err)
					return
				}
				break
			}
		}

		index = 0
		for index < n {

			if !bytes.Equal(buf[index+protocol.ReservedOffset:index+protocol.ReservedOffset+protocol.ReservedSize], protocol.ReservedBytes) {
				c.Logger().Error().Msgf(InvalidBufferContents.Error())
				break
			}

			decodedMessage := protocol.Message{
				From:          binary.BigEndian.Uint32(buf[index+protocol.FromOffset : index+protocol.FromOffset+protocol.FromSize]),
				To:            binary.BigEndian.Uint32(buf[index+protocol.ToOffset : index+protocol.ToOffset+protocol.ToSize]),
				Id:            binary.BigEndian.Uint64(buf[index+protocol.IdOffset : index+protocol.IdOffset+protocol.IdSize]),
				Operation:     binary.BigEndian.Uint32(buf[index+protocol.OperationOffset : index+protocol.OperationOffset+protocol.OperationSize]),
				ContentLength: binary.BigEndian.Uint64(buf[index+protocol.ContentLengthOffset : index+protocol.ContentLengthOffset+protocol.ContentLengthSize]),
			}

			index += protocol.MessageSize

			if decodedMessage.Operation == STREAMCLOSE {
				c.streamConnMutex.RLock()
				streamConn := c.streamConns[decodedMessage.Id]
				c.streamConnMutex.RUnlock()
				if streamConn != nil {
					streamConn.closed.Store(true)
					c.streamConnMutex.Lock()
					delete(c.streamConns, decodedMessage.Id)
					c.streamConnMutex.Unlock()
				}
				goto READ
			}
			if decodedMessage.ContentLength > 0 {
				switch decodedMessage.Operation {
				case NEWSTREAM:
					c.streamConnMutex.RLock()
					streamConn := c.streamConns[decodedMessage.Id]
					c.streamConnMutex.RUnlock()
					if streamConn == nil {
						streamConn = c.NewStream(decodedMessage.Id)
						c.streamConnMutex.Lock()
						c.streamConns[decodedMessage.Id] = streamConn
						c.streamConnMutex.Unlock()
						select {
						case c.streamConnCh <- streamConn:
						default:
						}
					}
					if n-index < int(decodedMessage.ContentLength) {
						streamConn.incomingBuffer.Lock()
						for streamConn.incomingBuffer.buffer.Cap()-streamConn.incomingBuffer.buffer.Len() < int(decodedMessage.ContentLength) {
							streamConn.incomingBuffer.buffer.Grow(DefaultBufferSize)
						}
						cp, err := streamConn.incomingBuffer.buffer.Write(buf[index:n])
						if err != nil {
							c.Logger().Debug().Msgf(errors.WithContext(err, WRITE).Error())
							_ = c.closeWithError(err)
							return
						}
						min := int64(int(decodedMessage.ContentLength) - cp)
						index = n
						_, err = io.CopyN(streamConn.incomingBuffer.buffer, c.conn, min)
						if err != nil {
							c.Logger().Debug().Msgf(errors.WithContext(err, WRITE).Error())
							_ = c.closeWithError(err)
							return
						}
						streamConn.incomingBuffer.Unlock()
					} else {
						streamConn.incomingBuffer.Lock()
						for streamConn.incomingBuffer.buffer.Cap()-streamConn.incomingBuffer.buffer.Len() < int(decodedMessage.ContentLength) {
							streamConn.incomingBuffer.buffer.Grow(DefaultBufferSize)
						}
						cp, err := streamConn.incomingBuffer.buffer.Write(buf[index : index+int(decodedMessage.ContentLength)])
						if err != nil {
							c.Logger().Debug().Msgf(errors.WithContext(err, WRITE).Error())
							_ = c.closeWithError(err)
							return
						}
						streamConn.incomingBuffer.Unlock()
						index += cp
					}
				default:
					readContent := make([]byte, decodedMessage.ContentLength)
					if n-index < int(decodedMessage.ContentLength) {
						for cap(buf) < int(decodedMessage.ContentLength) {
							buf = append(buf[:cap(buf)], 0)
							buf = buf[:cap(buf)]
						}
						cp := copy(readContent, buf[index:n])
						buf = buf[:cap(buf)]
						min := int(decodedMessage.ContentLength) - cp
						if len(buf) < min {
							_ = c.closeWithError(InvalidBufferLength)
							return
						}
						n = 0
						for n < min {
							var nn int
							nn, err = c.conn.Read(buf[n:])
							n += nn
							if err != nil {
								if n < min {
									_ = c.closeWithError(err)
									return
								}
								break
							}
						}
						copy(readContent[cp:], buf[:min])
						index = min
					} else {
						copy(readContent, buf[index:index+int(decodedMessage.ContentLength)])
						index += int(decodedMessage.ContentLength)
					}
					err = c.incomingMessages.Push(&protocol.Packet{
						Message: &decodedMessage,
						Content: &readContent,
					})
					if err != nil {
						c.Logger().Debug().Msgf(errors.WithContext(err, PUSH).Error())
						_ = c.closeWithError(err)
						return
					}
				}
			} else {
				err = c.incomingMessages.Push(&protocol.Packet{
					Message: &decodedMessage,
					Content: nil,
				})
				if err != nil {
					c.Logger().Debug().Msgf(errors.WithContext(err, PUSH).Error())
					_ = c.closeWithError(err)
					return
				}
			}
		READ:
			if n == index {
				index = 0
				buf = buf[:cap(buf)]
				if len(buf) < protocol.MessageSize {
					_ = c.closeWithError(InvalidBufferLength)
					break
				}
				n = 0
				for n < protocol.MessageSize {
					var nn int
					nn, err = c.conn.Read(buf[n:])
					n += nn
					if err != nil {
						if n < protocol.MessageSize {
							_ = c.closeWithError(err)
							return
						}
						break
					}
				}
			} else if n-index < protocol.MessageSize {
				copy(buf, buf[index:n])
				n -= index
				index = n

				buf = buf[:cap(buf)]
				min := protocol.MessageSize - index
				if len(buf) < min {
					_ = c.closeWithError(InvalidBufferLength)
					break
				}
				n = 0
				for n < min {
					var nn int
					nn, err = c.conn.Read(buf[index+n:])
					n += nn
					if err != nil {
						if n < min {
							_ = c.closeWithError(err)
							return
						}
						break
					}
				}
				n += index
				index = 0
			}
		}
	}
}

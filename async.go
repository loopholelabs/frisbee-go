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
	"crypto/tls"
	"encoding/binary"
	"github.com/loopholelabs/frisbee/internal/protocol"
	"github.com/loopholelabs/frisbee/internal/ringbuffer"
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
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
	closed           *atomic.Bool
	writer           *bufio.Writer
	flusher          chan struct{}
	incomingMessages *ringbuffer.RingBuffer
	logger           *zerolog.Logger
	wg               sync.WaitGroup
	error            *atomic.Error
	pongCh           chan struct{}
	closeCh          chan struct{}
}

// ConnectAsync creates a new TCP connection (using net.Dial) and wraps it in a frisbee connection
func ConnectAsync(addr string, keepAlive time.Duration, logger *zerolog.Logger, TLSConfig *tls.Config) (*Async, error) {
	var conn net.Conn
	var err error

	if TLSConfig != nil {
		conn, err = tls.Dial("tcp", addr, TLSConfig)
	} else {
		conn, err = net.Dial("tcp", addr)
		_ = conn.(*net.TCPConn).SetKeepAlive(true)
		_ = conn.(*net.TCPConn).SetKeepAlivePeriod(keepAlive)
	}

	if err != nil {
		return nil, err
	}

	return NewAsync(conn, logger), nil
}

// NewAsync takes an existing net.Conn object and wraps it in a frisbee connection
func NewAsync(c net.Conn, logger *zerolog.Logger) (conn *Async) {
	conn = &Async{
		conn:             c,
		closed:           atomic.NewBool(false),
		writer:           bufio.NewWriterSize(c, DefaultBufferSize),
		incomingMessages: ringbuffer.NewRingBuffer(DefaultBufferSize),
		flusher:          make(chan struct{}, 1024),
		logger:           logger,
		error:            atomic.NewError(nil),
		pongCh:           make(chan struct{}, 1),
		closeCh:          make(chan struct{}, 1),
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
	if c.closed.Load() {
		return ConnectionClosed
	}
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying net.Conn
func (c *Async) SetReadDeadline(t time.Time) error {
	if c.closed.Load() {
		return ConnectionClosed
	}
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying net.Conn
func (c *Async) SetWriteDeadline(t time.Time) error {
	if c.closed.Load() {
		return ConnectionClosed
	}
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

// CloseChannel returns a channel that can be listened to for a close event on a frisbee connection
func (c *Async) CloseChannel() <-chan struct{} {
	return c.closeCh
}

// WritePacket takes a packet.Packet and queues it up to send asynchronously.
//
// If message.ContentLength == 0, then the content array must be nil. Otherwise, it is required that message.ContentLength == len(content).
func (c *Async) WritePacket(p *packet.Packet) error {
	if int(p.Metadata.ContentLength) != len(p.Content) {
		return InvalidContentLength
	}

	var encodedMessage [protocol.MessageSize]byte

	binary.BigEndian.PutUint16(encodedMessage[protocol.IdOffset:protocol.IdOffset+protocol.IdSize], p.Metadata.Id)
	binary.BigEndian.PutUint16(encodedMessage[protocol.OperationOffset:protocol.OperationOffset+protocol.OperationSize], p.Metadata.Operation)
	binary.BigEndian.PutUint32(encodedMessage[protocol.ContentLengthOffset:protocol.ContentLengthOffset+protocol.ContentLengthSize], p.Metadata.ContentLength)

	c.Lock()
	if c.closed.Load() {
		c.Unlock()
		return ConnectionClosed
	}

	_, err := c.writer.Write(encodedMessage[:])
	if err != nil {
		c.Unlock()
		if c.closed.Load() {
			c.Logger().Error().Err(ConnectionClosed).Msg("error while writing encoded message")
			return ConnectionClosed
		}
		c.Logger().Error().Err(err).Msg("error while writing encoded message")
		return c.closeWithError(err)
	}
	if p.Metadata.ContentLength != 0 {
		if int(p.Metadata.ContentLength) > c.writer.Size() {
			err = c.SetWriteDeadline(time.Now().Add(defaultDeadline))
			if err != nil {
				c.Unlock()
				if c.closed.Load() {
					c.Logger().Error().Err(ConnectionClosed).Msg("error while writing message content")
					return ConnectionClosed
				}
				c.Logger().Error().Err(err).Msg("error while writing message content")
				return c.closeWithError(err)
			}
		}
		_, err = c.writer.Write(p.Content[:p.Metadata.ContentLength])
		if err != nil {
			c.Unlock()
			if c.closed.Load() {
				c.Logger().Error().Err(ConnectionClosed).Msg("error while writing message content")
				return ConnectionClosed
			}
			c.Logger().Error().Err(err).Msg("error while writing message content")
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

// ReadPacket is a blocking function that will wait until a Frisbee message is available and then return it (and its content).
// In the event that the connection is closed, ReadMessage will return an error.
func (c *Async) ReadPacket() (*packet.Packet, error) {
	if c.closed.Load() {
		return nil, ConnectionClosed
	}

	readPacket, err := c.incomingMessages.Pop()
	if err != nil {
		if c.closed.Load() {
			c.Logger().Error().Err(ConnectionClosed).Msg("error while popping from message queue")
			return nil, ConnectionClosed
		}
		c.Logger().Error().Err(err).Msg("error while popping from message queue")
		return nil, err
	}

	return readPacket, nil
}

// Flush allows for synchronous messaging by flushing the message buffer and instantly sending messages
func (c *Async) Flush() error {
	c.Lock()
	if c.closed.Load() {
		c.Unlock()
		return ConnectionClosed
	}
	if c.writer.Buffered() > 0 {
		err := c.SetWriteDeadline(time.Now().Add(defaultDeadline))
		if err != nil {
			c.Unlock()
			return c.closeWithError(err)
		}
		err = c.writer.Flush()
		if err != nil {
			c.Unlock()
			c.Logger().Err(err).Msg("error while flushing data")
			return c.closeWithError(err)
		}
	}
	c.Unlock()
	return nil
}

// WriteBufferSize returns the size of the underlying message buffer (used for internal message handling and for heartbeat logic)
func (c *Async) WriteBufferSize() int {
	c.Lock()
	if c.closed.Load() {
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

// Error returns the error that caused the frisbee.Async connection to close
func (c *Async) Error() error {
	return c.error.Load()
}

// Closed returns whether the frisbee.Async connection is closed
func (c *Async) Closed() bool {
	return c.closed.Load()
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
	_ = c.conn.SetDeadline(pastTime)
	c.wg.Wait()
	_ = c.conn.SetDeadline(emptyTime)
	close(c.closeCh)
	c.Logger().Debug().Msg("error channel closed, goroutines killed")
}

func (c *Async) close() error {
	if c.closed.CAS(false, true) {
		c.Logger().Debug().Msg("connection close called, killing goroutines")
		c.killGoroutines()
		c.Lock()
		if c.writer.Buffered() > 0 {
			_ = c.conn.SetWriteDeadline(emptyTime)
			_ = c.writer.Flush()
		}
		c.Unlock()
		return nil
	}
	return ConnectionClosed
}

func (c *Async) closeWithError(err error) error {
	closeError := c.close()
	if closeError != nil {
		c.Logger().Debug().Err(closeError).Msgf("attempted to close connection with error `%s`, but got error while closing", err)
		return closeError
	}
	c.error.Store(err)
	_ = c.conn.Close()
	return err
}

func (c *Async) flushLoop() {
	for {
		if _, ok := <-c.flusher; !ok {
			c.wg.Done()
			return
		}
		err := c.Flush()
		if err != nil {
			c.wg.Done()
			_ = c.closeWithError(err)
			return
		}
	}
}

func (c *Async) waitForPONG() {
	timer := time.NewTimer(defaultDeadline)
	defer timer.Stop()
	select {
	case <-timer.C:
		c.Logger().Error().Err(os.ErrDeadlineExceeded).Msg("timed out waiting for PONG, connection is not alive")
		c.wg.Done()
		_ = c.closeWithError(os.ErrDeadlineExceeded)
	case <-c.pongCh:
		c.wg.Done()
		c.Logger().Debug().Msg("PONG message received on time, connection is alive")
	}
}

func (c *Async) handleTimeout() error {
	if c.closed.Load() {
		return ConnectionClosed
	}

	c.Logger().Debug().Msg("Handling Timeout Using PING Packet")
	err := c.WritePacket(PINGPacket)
	if err != nil {
		return err
	}

	err = c.Flush()
	if err != nil {
		return err
	}

	c.Logger().Debug().Msg("PING Packet sent successfully, will wait for PONG in a separate thread")
	c.wg.Add(1)
	go c.waitForPONG()

	return nil
}

func (c *Async) readLoop() {
	buf := make([]byte, DefaultBufferSize)
	var index int
	for {
		buf = buf[:cap(buf)]
		if len(buf) < protocol.MessageSize {
			c.Logger().Debug().Err(InvalidBufferLength).Msg("error during read loop, calling closeWithError")
			c.wg.Done()
			_ = c.closeWithError(InvalidBufferLength)
			return
		}

		var n int
		var err error
		for n < protocol.MessageSize {
			var nn int
			err = c.SetReadDeadline(time.Now().Add(defaultDeadline))
			if err != nil {
				c.Logger().Debug().Err(err).Msg("error setting read deadline during read loop, calling closeWithError")
				c.wg.Done()
				_ = c.closeWithError(err)
				return
			}
			nn, err = c.conn.Read(buf[n:])
			n += nn
			if err != nil {
				if n < protocol.MessageSize {
					if errors.Is(err, os.ErrDeadlineExceeded) {
						err = c.handleTimeout()
						if err != nil {
							c.wg.Done()
							_ = c.closeWithError(err)
							return
						}
						continue
					}
					c.wg.Done()
					_ = c.closeWithError(err)
					return
				}
				break
			}
		}

		index = 0
		for index < n {
			p := packet.Get()
			p.Metadata.Id = binary.BigEndian.Uint16(buf[index+protocol.IdOffset : index+protocol.IdOffset+protocol.IdSize])
			p.Metadata.Operation = binary.BigEndian.Uint16(buf[index+protocol.OperationOffset : index+protocol.OperationOffset+protocol.OperationSize])
			p.Metadata.ContentLength = binary.BigEndian.Uint32(buf[index+protocol.ContentLengthOffset : index+protocol.ContentLengthOffset+protocol.ContentLengthSize])
			index += protocol.MessageSize

			switch p.Metadata.Operation {
			case PING:
				c.Logger().Debug().Msg("PING Packet received by read loop, sending back PONG message")
				err = c.WritePacket(PONGPacket)
				if err != nil {
					c.wg.Done()
					_ = c.closeWithError(err)
					return
				}
			case PONG:
				c.Logger().Debug().Msg("PONG Packet received by read loop")
				select {
				case c.pongCh <- struct{}{}:
				default:
				}
			default:
				if p.Metadata.ContentLength > 0 {
					//for cap(p.Content) < int(p.Metadata.ContentLength) {
					//	p.Content = append(p.Content[:cap(p.Content)], 0)
					//}
					//p.Content = p.Content[:p.Metadata.ContentLength]
					if n-index < int(p.Metadata.ContentLength) {
						for cap(buf) < int(p.Metadata.ContentLength) {
							buf = append(buf[:cap(buf)], 0)
						}
						buf = buf[:cap(buf)]
						cp := p.Write(buf[index:n])
						//cp := copy(p.Content[0:], buf[index:n])
						min := int(p.Metadata.ContentLength) - cp
						if len(buf) < min {
							c.wg.Done()
							_ = c.closeWithError(InvalidBufferLength)
							return
						}
						n = 0
						err = c.SetReadDeadline(emptyTime)
						if err != nil {
							c.wg.Done()
							_ = c.closeWithError(err)
							return
						}
						for n < min {
							var nn int
							nn, err = c.conn.Read(buf[n:])
							n += nn
							if err != nil {
								if n < min {
									c.wg.Done()
									_ = c.closeWithError(err)
									return
								}
								break
							}
						}
						p.Content = append(p.Content, buf[:min]...)
						p.Content = p.Content[:p.Metadata.ContentLength]
						//copy(p.Content[cp:], buf[:min])
						index = min
					} else {
						index += p.Write(buf[index : index+int(p.Metadata.ContentLength)])
						//index += copy(p.Content[0:], buf[index:index+int(p.Metadata.ContentLength)])
					}
					err = c.incomingMessages.Push(p)
					if err != nil {
						c.Logger().Error().Err(err).Msg("error while pushing to incoming message queue")
						c.wg.Done()
						_ = c.closeWithError(err)
						return
					}
				} else {
					err = c.incomingMessages.Push(p)
					if err != nil {
						c.Logger().Error().Err(err).Msg("error while pushing to incoming message queue")
						c.wg.Done()
						_ = c.closeWithError(err)
						return
					}
				}
			}
			if n == index {
				index = 0
				buf = buf[:cap(buf)]
				if len(buf) < protocol.MessageSize {
					c.wg.Done()
					_ = c.closeWithError(InvalidBufferLength)
					return
				}
				n = 0
				for n < protocol.MessageSize {
					var nn int
					err = c.SetReadDeadline(time.Now().Add(defaultDeadline))
					if err != nil {
						c.wg.Done()
						_ = c.closeWithError(err)
						return
					}
					nn, err = c.conn.Read(buf[n:])
					n += nn
					if err != nil {
						if n < protocol.MessageSize {
							if errors.Is(err, os.ErrDeadlineExceeded) {
								err = c.handleTimeout()
								if err != nil {
									c.wg.Done()
									_ = c.closeWithError(err)
									return
								}
								continue
							}
							c.wg.Done()
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
					c.wg.Done()
					_ = c.closeWithError(InvalidBufferLength)
					return
				}
				n = 0
				for n < min {
					var nn int
					err = c.SetReadDeadline(time.Now().Add(defaultDeadline))
					if err != nil {
						c.wg.Done()
						_ = c.closeWithError(err)
						return
					}
					nn, err = c.conn.Read(buf[index+n:])
					n += nn
					if err != nil {
						if n < min {
							if errors.Is(err, os.ErrDeadlineExceeded) {
								err = c.handleTimeout()
								if err != nil {
									c.wg.Done()
									_ = c.closeWithError(err)
									return
								}
								continue
							}
							c.wg.Done()
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

// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/loopholelabs/common/pkg/queue"
	"github.com/loopholelabs/logging/loggers/noop"
	"github.com/loopholelabs/logging/types"

	"github.com/loopholelabs/frisbee-go/internal/dialer"
	"github.com/loopholelabs/frisbee-go/pkg/metadata"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
)

// Async is the underlying asynchronous frisbee connection which has extremely efficient read and write logic and
// can handle the specific frisbee requirements. This is not meant to be used on its own, and instead is
// meant to be used by frisbee client and server implementations
type Async struct {
	sync.Mutex
	conn               net.Conn
	closed             atomic.Bool
	writer             *bufio.Writer
	flushCh            chan struct{}
	closeCh            chan struct{}
	incoming           *queue.Circular[packet.Packet, *packet.Packet]
	staleMu            sync.Mutex
	stale              []*packet.Packet
	logger             types.Logger
	wg                 sync.WaitGroup
	error              atomic.Value
	streamsMu          sync.Mutex
	streams            map[uint16]*Stream
	newStreamHandlerMu sync.Mutex
	newStreamHandler   NewStreamHandler
}

// ConnectAsync creates a new TCP connection (using net.Dial) and wraps it in a frisbee connection
func ConnectAsync(addr string, keepAlive time.Duration, logger types.Logger, TLSConfig *tls.Config, streamHandler ...NewStreamHandler) (*Async, error) {
	var conn net.Conn
	var err error

	d := dialer.NewRetry()

	if TLSConfig != nil {
		conn, err = d.DialTLS("tcp", addr, TLSConfig)
	} else {
		conn, err = d.Dial("tcp", addr)
		if err == nil {
			_ = conn.(*net.TCPConn).SetKeepAlive(true)
			_ = conn.(*net.TCPConn).SetKeepAlivePeriod(keepAlive)
		}
	}

	if err != nil {
		return nil, err
	}

	return NewAsync(conn, logger, streamHandler...), nil
}

// NewAsync takes an existing net.Conn object and wraps it in a frisbee connection
func NewAsync(c net.Conn, logger types.Logger, streamHandler ...NewStreamHandler) (conn *Async) {
	conn = &Async{
		conn:     c,
		writer:   bufio.NewWriterSize(c, DefaultBufferSize),
		incoming: queue.NewCircular[packet.Packet, *packet.Packet](DefaultBufferSize),
		flushCh:  make(chan struct{}, 3),
		closeCh:  make(chan struct{}),
		streams:  make(map[uint16]*Stream),
		logger:   logger,
	}

	if logger == nil {
		conn.logger = noop.New(types.InfoLevel)
	}

	if len(streamHandler) > 0 {
		conn.newStreamHandler = streamHandler[0]
	}

	conn.wg.Add(3)
	go conn.flushLoop()
	go conn.readLoop()
	go conn.pingLoop()

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
	return emptyState, NotTLSConnectionError
}

// Handshake performs the tls.Handshake() of a *tls.Conn
// if the connection is not *tls.Conn then the NotTLSConnectionError is returned
func (c *Async) Handshake() error {
	if tlsConn, ok := c.conn.(*tls.Conn); ok {
		return tlsConn.Handshake()
	}
	return NotTLSConnectionError
}

// HandshakeContext performs the tls.HandshakeContext() of a *tls.Conn
// if the connection is not *tls.Conn then the NotTLSConnectionError is returned
func (c *Async) HandshakeContext(ctx context.Context) error {
	if tlsConn, ok := c.conn.(*tls.Conn); ok {
		return tlsConn.HandshakeContext(ctx)
	}
	return NotTLSConnectionError
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
// If packet.Metadata.ContentLength == 0, then the content array's length must be 0. Otherwise, it is required that packet.Metadata.ContentLength == len(content).
func (c *Async) WritePacket(p *packet.Packet) error {
	if p.Metadata.Operation <= RESERVED9 {
		return InvalidOperation
	}
	return c.writePacket(p, true)
}

// ReadPacket is a blocking function that will wait until a Frisbee packet is available and then return it (and its content).
// In the event that the connection is closed, ReadPacket will return an error.
func (c *Async) ReadPacket() (*packet.Packet, error) {
	if c.closed.Load() {
		c.staleMu.Lock()
		if len(c.stale) > 0 {
			var p *packet.Packet
			p, c.stale = c.stale[0], c.stale[1:]
			c.staleMu.Unlock()
			return p, nil
		}
		c.staleMu.Unlock()
		c.Logger().Debug().Err(ConnectionClosed).Msg("error while popping from packet queue")
		return nil, ConnectionClosed
	}

	readPacket, err := c.incoming.Pop()
	if err != nil {
		if c.closed.Load() {
			c.staleMu.Lock()
			if len(c.stale) > 0 {
				var p *packet.Packet
				p, c.stale = c.stale[0], c.stale[1:]
				c.staleMu.Unlock()
				return p, nil
			}
			c.staleMu.Unlock()
			c.Logger().Debug().Err(ConnectionClosed).Msg("error while popping from packet queue")
			return nil, ConnectionClosed
		}
		c.Logger().Debug().Err(err).Msg("error while popping from packet queue")
		return nil, err
	}

	return readPacket, nil
}

// Flush allows for synchronous messaging by flushing the write buffer and instantly sending packets
func (c *Async) Flush() error {
	err := c.flush()
	if err != nil {
		return c.closeWithError(err)
	}
	return nil
}

// WriteBufferSize returns the size of the underlying write buffer (used for internal packet handling and for heartbeat logic)
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
func (c *Async) Logger() types.Logger {
	return c.logger
}

// Error returns the error that caused the frisbee.Async connection to close
func (c *Async) Error() error {
	err := c.error.Load()
	if err == nil {
		return nil
	}
	return err.(error)
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

// NewStream returns a new stream that can be used to send and receive packets
func (c *Async) NewStream(id uint16) (stream *Stream) {
	c.streamsMu.Lock()
	if stream = c.streams[id]; stream == nil {
		stream = newStream(id, c)
		c.streams[id] = stream
	}
	c.streamsMu.Unlock()
	return
}

// SetNewStreamHandler sets the callback handler for new streams.
//
// It's important to note that this handler is called for new streams and if it is
// not set then stream packets will be dropped.
//
// It's also important to note that the handler itself is called in its own goroutine to
// avoid blocking the read lop. This means that the handler must be thread-safe.`
func (c *Async) SetNewStreamHandler(handler NewStreamHandler) {
	c.newStreamHandlerMu.Lock()
	c.newStreamHandler = handler
	c.newStreamHandlerMu.Unlock()
}

// Close closes the frisbee connection gracefully
func (c *Async) Close() error {
	err := c.close()
	if err != nil && errors.Is(err, ConnectionClosed) {
		return nil
	}
	_ = c.conn.Close()
	return err
}

// write packet is the internal write packet function that does not check for reserved operations.
func (c *Async) writePacket(p *packet.Packet, closeOnErr bool) error {
	if int(p.Metadata.ContentLength) != p.Content.Len() {
		return InvalidContentLength
	}

	encodedMetadata := metadata.GetBuffer()
	binary.BigEndian.PutUint16(encodedMetadata[metadata.IdOffset:metadata.IdOffset+metadata.IdSize], p.Metadata.Id)
	binary.BigEndian.PutUint16(encodedMetadata[metadata.OperationOffset:metadata.OperationOffset+metadata.OperationSize], p.Metadata.Operation)
	binary.BigEndian.PutUint32(encodedMetadata[metadata.ContentLengthOffset:metadata.ContentLengthOffset+metadata.ContentLengthSize], p.Metadata.ContentLength)

	c.Lock()
	if c.closed.Load() {
		c.Unlock()
		return ConnectionClosed
	}
	err := c.conn.SetWriteDeadline(time.Now().Add(DefaultDeadline))
	if err != nil {
		c.Unlock()
		if c.closed.Load() {
			c.Logger().Debug().Err(ConnectionClosed).Uint16("Packet ID", p.Metadata.Id).Msg("error while setting write deadline before writing packet")
			return ConnectionClosed
		}
		c.Logger().Debug().Err(err).Uint16("Packet ID", p.Metadata.Id).Msg("error while setting write deadline before writing packet")
		if closeOnErr {
			return c.closeWithError(err)
		}
		return err
	}
	_, err = c.writer.Write(encodedMetadata[:])
	metadata.PutBuffer(encodedMetadata)
	if err != nil {
		c.Unlock()
		if c.closed.Load() {
			c.Logger().Debug().Err(ConnectionClosed).Uint16("Packet ID", p.Metadata.Id).Msg("error while writing encoded metadata")
			return ConnectionClosed
		}
		c.Logger().Debug().Err(err).Uint16("Packet ID", p.Metadata.Id).Msg("error while writing encoded metadata")
		if closeOnErr {
			return c.closeWithError(err)
		}
		return err
	}
	if p.Metadata.ContentLength != 0 {
		_, err = c.writer.Write(p.Content.Bytes()[:p.Metadata.ContentLength])
		if err != nil {
			c.Unlock()
			if c.closed.Load() {
				c.Logger().Debug().Err(ConnectionClosed).Uint16("Packet ID", p.Metadata.Id).Msg("error while writing packet content")
				return ConnectionClosed
			}
			c.Logger().Debug().Err(err).Uint16("Packet ID", p.Metadata.Id).Msg("error while writing packet content")
			if closeOnErr {
				return c.closeWithError(err)
			}
			return err
		}
	}

	if len(c.flushCh) == 0 {
		select {
		case c.flushCh <- struct{}{}:
		default:
		}
	}

	c.Unlock()

	return nil
}

// flush is an internal function for flushing data from the write buffer, however
// it is unique in that it does not call closeWithError (and so does not try and close the underlying connection)
// when it encounters an error, and instead leaves that responsibility to its parent caller
func (c *Async) flush() error {
	c.Lock()
	if c.closed.Load() {
		c.Unlock()
		return ConnectionClosed
	}
	if c.writer.Buffered() > 0 {
		err := c.conn.SetWriteDeadline(time.Now().Add(DefaultDeadline))
		if err != nil {
			c.Unlock()
			return err
		}
		err = c.writer.Flush()
		if err != nil {
			c.Unlock()
			c.Logger().Error().Err(err).Msg("error while flushing data")
			return err
		}
	}
	c.Unlock()
	return nil
}

func (c *Async) close() error {
	c.staleMu.Lock()
	c.streamsMu.Lock()
	if c.closed.CompareAndSwap(false, true) {
		c.Logger().Debug().Msg("connection close called, killing goroutines")
		c.Lock()
		c.incoming.Close()
		close(c.closeCh)
		close(c.flushCh)
		c.Unlock()
		_ = c.conn.SetDeadline(pastTime)
		c.wg.Wait()
		_ = c.conn.SetDeadline(emptyTime)
		c.stale = c.incoming.Drain()
		c.staleMu.Unlock()
		for _, stream := range c.streams {
			_ = stream.closeSend(false)
		}
		c.streamsMu.Unlock()
		c.Lock()
		if c.writer.Buffered() > 0 {
			_ = c.conn.SetWriteDeadline(time.Now().Add(DefaultDeadline))
			_ = c.writer.Flush()
			_ = c.conn.SetWriteDeadline(emptyTime)
		}
		c.Unlock()
		return nil
	}
	c.staleMu.Unlock()
	c.streamsMu.Unlock()
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
	var err error
	for {
		if _, ok := <-c.flushCh; !ok {
			c.wg.Done()
			return
		}
		err = c.flush()
		if err != nil {
			c.wg.Done()
			_ = c.closeWithError(err)
			return
		}
	}
}

func (c *Async) pingLoop() {
	ticker := time.NewTicker(DefaultPingInterval)
	defer ticker.Stop()
	var err error
	for {
		select {
		case <-c.closeCh:
			c.wg.Done()
			return
		case <-ticker.C:
			err = c.writePacket(PINGPacket, false)
			if err != nil {
				c.wg.Done()
				_ = c.closeWithError(err)
				return
			}
		}
	}
}

func (c *Async) readLoop() {
	buf := make([]byte, DefaultBufferSize)
	var index int
	var stream *Stream
	var isStream bool
	var newStreamHandler NewStreamHandler
	for {
		buf = buf[:cap(buf)]
		if len(buf) < metadata.Size {
			c.Logger().Debug().Err(InvalidBufferLength).Msg("error during read loop, calling closeWithError")
			c.wg.Done()
			_ = c.closeWithError(InvalidBufferLength)
			return
		}

		var n int
		var err error
		for n < metadata.Size {
			var nn int
			err = c.conn.SetReadDeadline(time.Now().Add(DefaultDeadline))
			if err != nil {
				c.Logger().Debug().Err(err).Msg("error setting read deadline during read loop, calling closeWithError")
				c.wg.Done()
				_ = c.closeWithError(err)
				return
			}
			nn, err = c.conn.Read(buf[n:])
			n += nn
			if err != nil {
				if n < metadata.Size {
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
			p.Metadata.Id = binary.BigEndian.Uint16(buf[index+metadata.IdOffset : index+metadata.IdOffset+metadata.IdSize])
			p.Metadata.Operation = binary.BigEndian.Uint16(buf[index+metadata.OperationOffset : index+metadata.OperationOffset+metadata.OperationSize])
			p.Metadata.ContentLength = binary.BigEndian.Uint32(buf[index+metadata.ContentLengthOffset : index+metadata.ContentLengthOffset+metadata.ContentLengthSize])
			index += metadata.Size

			switch p.Metadata.Operation {
			case PING:
				c.Logger().Debug().Msg("PING Packet received by read loop, sending back PONG packet")
				err = c.writePacket(PONGPacket, false)
				if err != nil {
					c.wg.Done()
					_ = c.closeWithError(err)
					return
				}
				packet.Put(p)
			case PONG:
				c.Logger().Debug().Msg("PONG Packet received by read loop")
				packet.Put(p)
			case STREAM:
				c.Logger().Debug().Msg("STREAM Packet received by read loop")
				isStream = true
				c.newStreamHandlerMu.Lock()
				newStreamHandler = c.newStreamHandler
				c.newStreamHandlerMu.Unlock()
				if newStreamHandler != nil || p.Metadata.ContentLength == 0 {
					c.streamsMu.Lock()
					stream = c.streams[p.Metadata.Id]
					c.streamsMu.Unlock()
				}
				fallthrough
			default:
				if p.Metadata.ContentLength > 0 {
					if n-index < int(p.Metadata.ContentLength) {
						minSize := int(p.Metadata.ContentLength) - p.Content.Write(buf[index:n])
						n = 0
						for cap(buf) < minSize {
							buf = append(buf[:cap(buf)], 0)
						}
						buf = buf[:cap(buf)]
						for n < minSize {
							var nn int
							err = c.conn.SetReadDeadline(time.Now().Add(DefaultDeadline))
							if err != nil {
								c.wg.Done()
								_ = c.closeWithError(err)
								return
							}
							nn, err = c.conn.Read(buf[n:])
							n += nn
							if err != nil {
								if n < minSize {
									c.wg.Done()
									_ = c.closeWithError(err)
									return
								}
								break
							}
						}
						p.Content.Write(buf[:minSize])
						index = minSize
					} else {
						index += p.Content.Write(buf[index : index+int(p.Metadata.ContentLength)])
					}
				}
				if !isStream {
					err = c.incoming.Push(p)
					if err != nil {
						c.Logger().Debug().Err(err).Msg("error while pushing to incoming packet queue")
						c.wg.Done()
						_ = c.closeWithError(err)
						return
					}
				} else {
					if p.Metadata.ContentLength == 0 {
						if stream != nil {
							stream.close()
							c.streamsMu.Lock()
							delete(c.streams, p.Metadata.Id)
							c.streamsMu.Unlock()
						}
						packet.Put(p)
					} else {
						if newStreamHandler == nil {
							c.Logger().Debug().Msg("STREAM Packet discarded by read loop")
							packet.Put(p)
						} else {
							if stream == nil {
								stream = newStream(p.Metadata.Id, c)
								c.streamsMu.Lock()
								c.streams[p.Metadata.Id] = stream
								c.streamsMu.Unlock()
								go newStreamHandler(stream)
							}
							err = stream.queue.Push(p)
							if err != nil {
								c.Logger().Debug().Err(err).Msg("error while pushing to a stream queue packet queue")
								c.wg.Done()
								_ = c.closeWithError(err)
								return
							}
						}
					}
				}
				newStreamHandler = nil
				stream = nil
				isStream = false
			}
			if n == index {
				index = 0
				buf = buf[:cap(buf)]
				if len(buf) < metadata.Size {
					c.wg.Done()
					_ = c.closeWithError(InvalidBufferLength)
					return
				}
				n = 0
				for n < metadata.Size {
					var nn int
					err = c.conn.SetReadDeadline(time.Now().Add(DefaultDeadline))
					if err != nil {
						c.wg.Done()
						_ = c.closeWithError(err)
						return
					}
					nn, err = c.conn.Read(buf[n:])
					n += nn
					if err != nil {
						if n < metadata.Size {
							c.wg.Done()
							_ = c.closeWithError(err)
							return
						}
						break
					}
				}
			} else if n-index < metadata.Size {
				copy(buf, buf[index:n])
				n -= index
				index = n

				buf = buf[:cap(buf)]
				minSize := metadata.Size - index
				if len(buf) < minSize {
					c.wg.Done()
					_ = c.closeWithError(InvalidBufferLength)
					return
				}
				n = 0
				for n < minSize {
					var nn int
					err = c.conn.SetReadDeadline(time.Now().Add(DefaultDeadline))
					if err != nil {
						c.wg.Done()
						_ = c.closeWithError(err)
						return
					}
					nn, err = c.conn.Read(buf[index+n:])
					n += nn
					if err != nil {
						if n < minSize {
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

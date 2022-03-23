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
	"context"
	"crypto/tls"
	"encoding/binary"
	"github.com/loopholelabs/frisbee/pkg/metadata"
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"io"
	"net"
	"sync"
	"time"
)

// Sync is the underlying synchronous frisbee connection which has extremely efficient read and write logic and
// can handle the specific frisbee requirements. This is not meant to be used on its own, and instead is
// meant to be used by frisbee client and server implementations
type Sync struct {
	sync.Mutex
	conn   net.Conn
	closed *atomic.Bool
	logger *zerolog.Logger
	error  *atomic.Error
	ctxMu  sync.RWMutex
	ctx    context.Context
}

// ConnectSync creates a new TCP connection (using net.Dial) and wraps it in a frisbee connection
func ConnectSync(addr string, keepAlive time.Duration, logger *zerolog.Logger, TLSConfig *tls.Config) (*Sync, error) {
	var conn net.Conn
	var err error

	if TLSConfig != nil {
		conn, err = tls.Dial("tcp", addr, TLSConfig)
	} else {
		conn, err = net.Dial("tcp", addr)
		if err == nil {
			_ = conn.(*net.TCPConn).SetKeepAlive(true)
			_ = conn.(*net.TCPConn).SetKeepAlivePeriod(keepAlive)
		}
	}

	if err != nil {
		return nil, err
	}

	return NewSync(conn, logger), nil
}

// NewSync takes an existing net.Conn object and wraps it in a frisbee connection
func NewSync(c net.Conn, logger *zerolog.Logger) (conn *Sync) {
	conn = &Sync{
		conn:   c,
		closed: atomic.NewBool(false),
		logger: logger,
		error:  atomic.NewError(nil),
	}

	if logger == nil {
		conn.logger = &defaultLogger
	}
	return
}

// SetDeadline sets the read and write deadline on the underlying net.Conn
func (c *Sync) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying net.Conn
func (c *Sync) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying net.Conn
func (c *Sync) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// ConnectionState returns the tls.ConnectionState of a *tls.Conn
// if the connection is not *tls.Conn then the NotTLSConnectionError is returned
func (c *Sync) ConnectionState() (tls.ConnectionState, error) {
	if tlsConn, ok := c.conn.(*tls.Conn); ok {
		return tlsConn.ConnectionState(), nil
	}
	return emptyState, NotTLSConnectionError
}

// Handshake performs the tls.Handshake() of a *tls.Conn
// if the connection is not *tls.Conn then the NotTLSConnectionError is returned
func (c *Sync) Handshake() error {
	if tlsConn, ok := c.conn.(*tls.Conn); ok {
		return tlsConn.Handshake()
	}
	return NotTLSConnectionError
}

// HandshakeContext performs the tls.HandshakeContext() of a *tls.Conn
// if the connection is not *tls.Conn then the NotTLSConnectionError is returned
func (c *Sync) HandshakeContext(ctx context.Context) error {
	if tlsConn, ok := c.conn.(*tls.Conn); ok {
		return tlsConn.HandshakeContext(ctx) //trunk-ignore(golangci-lint/typecheck)
	}
	return NotTLSConnectionError
}

// LocalAddr returns the local address of the underlying net.Conn
func (c *Sync) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote address of the underlying net.Conn
func (c *Sync) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// WritePacket takes a packet.Packet and sends it synchronously.
//
// If packet.Metadata.ContentLength == 0, then the content array must be nil. Otherwise, it is required that packet.Metadata.ContentLength == len(content).
func (c *Sync) WritePacket(p *packet.Packet) error {
	if int(p.Metadata.ContentLength) != len(p.Content.B) {
		return InvalidContentLength
	}

	var encodedMetadata [metadata.Size]byte

	binary.BigEndian.PutUint16(encodedMetadata[metadata.IdOffset:metadata.IdOffset+metadata.IdSize], p.Metadata.Id)
	binary.BigEndian.PutUint16(encodedMetadata[metadata.OperationOffset:metadata.OperationOffset+metadata.OperationSize], p.Metadata.Operation)
	binary.BigEndian.PutUint32(encodedMetadata[metadata.ContentLengthOffset:metadata.ContentLengthOffset+metadata.ContentLengthSize], p.Metadata.ContentLength)

	c.Lock()
	if c.closed.Load() {
		c.Unlock()
		return ConnectionClosed
	}

	_, err := c.conn.Write(encodedMetadata[:])
	if err != nil {
		c.Unlock()
		if c.closed.Load() {
			c.Logger().Debug().Err(ConnectionClosed).Uint16("Packet ID", p.Metadata.Id).Msg("error while writing encoded metadata")
			return ConnectionClosed
		}
		c.Logger().Debug().Err(err).Uint16("Packet ID", p.Metadata.Id).Msg("error while writing encoded metadata")
		return c.closeWithError(err)
	}
	if p.Metadata.ContentLength != 0 {
		_, err = c.conn.Write(p.Content.B[:p.Metadata.ContentLength])
		if err != nil {
			c.Unlock()
			if c.closed.Load() {
				c.Logger().Debug().Err(ConnectionClosed).Uint16("Packet ID", p.Metadata.Id).Msg("error while writing encoded metadata")
				return ConnectionClosed
			}
			c.Logger().Debug().Err(err).Uint16("Packet ID", p.Metadata.Id).Msg("error while writing encoded metadata")
			return c.closeWithError(err)
		}
	}

	c.Unlock()
	return nil
}

// ReadPacket is a blocking function that will wait until a frisbee packet is available and then return it (and its content).
// In the event that the connection is closed, ReadPacket will return an error.
func (c *Sync) ReadPacket() (*packet.Packet, error) {
	if c.closed.Load() {
		return nil, ConnectionClosed
	}
	var encodedPacket [metadata.Size]byte

	_, err := io.ReadAtLeast(c.conn, encodedPacket[:], metadata.Size)
	if err != nil {
		if c.closed.Load() {
			c.Logger().Debug().Err(ConnectionClosed).Msg("error while reading from underlying net.Conn")
			return nil, ConnectionClosed
		}
		c.Logger().Debug().Err(err).Msg("error while reading from underlying net.Conn")
		return nil, c.closeWithError(err)
	}
	p := packet.Get()

	p.Metadata.Id = binary.BigEndian.Uint16(encodedPacket[metadata.IdOffset : metadata.IdOffset+metadata.IdSize])
	p.Metadata.Operation = binary.BigEndian.Uint16(encodedPacket[metadata.OperationOffset : metadata.OperationOffset+metadata.OperationSize])
	p.Metadata.ContentLength = binary.BigEndian.Uint32(encodedPacket[metadata.ContentLengthOffset : metadata.ContentLengthOffset+metadata.ContentLengthSize])

	if p.Metadata.ContentLength > 0 {
		for cap(p.Content.B) < int(p.Metadata.ContentLength) {
			p.Content.B = append(p.Content.B[:cap(p.Content.B)], 0)
		}
		p.Content.B = p.Content.B[:p.Metadata.ContentLength]
		_, err = io.ReadAtLeast(c.conn, p.Content.B[:], int(p.Metadata.ContentLength))
		if err != nil {
			if c.closed.Load() {
				c.Logger().Debug().Err(ConnectionClosed).Msg("error while reading from underlying net.Conn")
				return nil, ConnectionClosed
			}
			c.Logger().Debug().Err(err).Msg("error while reading from underlying net.Conn")
			return nil, c.closeWithError(err)
		}
	}

	return p, nil
}

// SetContext allows users to save a context within a connection
func (c *Sync) SetContext(ctx context.Context) {
	c.ctxMu.Lock()
	c.ctx = ctx
	c.ctxMu.Unlock()
}

// Context returns the saved context within the connection
func (c *Sync) Context() (ctx context.Context) {
	c.ctxMu.RLock()
	ctx = c.ctx
	c.ctxMu.RUnlock()
	return
}

// Logger returns the underlying logger of the frisbee connection
func (c *Sync) Logger() *zerolog.Logger {
	return c.logger
}

// Error returns the error that caused the frisbee.Sync to close or go into a paused state
func (c *Sync) Error() error {
	return c.error.Load()
}

// Raw shuts off all of frisbee's underlying functionality and converts the frisbee connection into a normal TCP connection (net.Conn)
func (c *Sync) Raw() net.Conn {
	_ = c.close()
	return c.conn
}

// Close closes the frisbee connection gracefully
func (c *Sync) Close() error {
	err := c.close()
	if errors.Is(err, ConnectionClosed) {
		return nil
	}
	_ = c.conn.Close()
	return err
}

func (c *Sync) close() error {
	if c.closed.CAS(false, true) {
		return nil
	}
	return ConnectionClosed
}

func (c *Sync) closeWithError(err error) error {
	closeError := c.close()
	if errors.Is(closeError, ConnectionClosed) {
		c.Logger().Debug().Err(err).Msg("attempted to close connection with error, but connection already closed")
		return ConnectionClosed
	} else {
		c.Logger().Debug().Err(err).Msgf("closing connection with error")
	}
	c.error.Store(err)
	_ = c.conn.Close()
	return err
}

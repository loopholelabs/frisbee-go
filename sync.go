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
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"github.com/loopholelabs/frisbee/internal/errors"
	"github.com/loopholelabs/frisbee/internal/protocol"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

// Sync is the underlying synchronous frisbee connection which has extremely efficient read and write logic and
// can handle the specific frisbee requirements. This is not meant to be used on its own, and instead is
// meant to be used by frisbee client and server implementations
type Sync struct {
	sync.Mutex
	conn   net.Conn
	state  *atomic.Int32
	logger *zerolog.Logger
	error  *atomic.Error
}

// ConnectSync creates a new TCP connection (using net.Dial) and wraps it in a frisbee connection
func ConnectSync(network string, addr string, keepAlive time.Duration, logger *zerolog.Logger, TLSConfig *tls.Config) (*Sync, error) {
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

  switch v := conn.(type) {
	case *net.TCPConn:
		_ = v.SetKeepAlive(true)
		_ = v.SetKeepAlivePeriod(keepAlive)
	}

	return NewSync(conn, logger), nil
}

// NewSync takes an existing net.Conn object and wraps it in a frisbee connection
func NewSync(c net.Conn, logger *zerolog.Logger) (conn *Sync) {
	conn = &Sync{
		conn:   c,
		state:  atomic.NewInt32(CONNECTED),
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
	return tls.ConnectionState{}, NotTLSConnectionError
}

// LocalAddr returns the local address of the underlying net.Conn
func (c *Sync) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote address of the underlying net.Conn
func (c *Sync) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// WriteMessage takes a frisbee.Message and some (optional) accompanying content, sends it synchronously.
//
// If message.ContentLength == 0, then the content array must be nil. Otherwise, it is required that message.ContentLength == len(content).
func (c *Sync) WriteMessage(message *Message, content *[]byte) error {
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

	_, err := c.conn.Write(encodedMessage[:])
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
		_, err = c.conn.Write(*content)
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

	c.Unlock()
	return nil
}

// ReadMessage is a blocking function that will wait until a frisbee message is available and then return it (and its content).
// In the event that the connection is closed, ReadMessage will return an error.
func (c *Sync) ReadMessage() (*Message, *[]byte, error) {
	if c.state.Load() != CONNECTED {
		return nil, nil, c.Error()
	}
	var encodedMessage [protocol.MessageSize]byte

	_, err := io.ReadAtLeast(c.conn, encodedMessage[:], protocol.MessageSize)
	if err != nil {
		if c.state.Load() != CONNECTED {
			err = c.Error()
			c.logger.Error().Msgf(errors.WithContext(err, POP).Error())
			return nil, nil, errors.WithContext(err, POP)
		}
		c.logger.Error().Msgf(errors.WithContext(err, POP).Error())
		return nil, nil, errors.WithContext(c.closeWithError(err), POP)
	}

	message := new(Message)

	if !bytes.Equal(encodedMessage[protocol.ReservedOffset:protocol.ReservedOffset+protocol.ReservedSize], protocol.ReservedBytes) {
		if c.state.Load() != CONNECTED {
			err = c.Error()
			c.logger.Error().Msgf(errors.WithContext(err, POP).Error())
			return nil, nil, errors.WithContext(err, POP)
		}
		c.logger.Error().Msgf(errors.WithContext(protocol.InvalidMessageVersion, POP).Error())
		return nil, nil, errors.WithContext(c.closeWithError(protocol.InvalidMessageVersion), POP)
	}

	message.From = binary.BigEndian.Uint32(encodedMessage[protocol.FromOffset : protocol.FromOffset+protocol.FromSize])
	message.To = binary.BigEndian.Uint32(encodedMessage[protocol.ToOffset : protocol.ToOffset+protocol.ToSize])
	message.Id = binary.BigEndian.Uint64(encodedMessage[protocol.IdOffset : protocol.IdOffset+protocol.IdSize])
	message.Operation = binary.BigEndian.Uint32(encodedMessage[protocol.OperationOffset : protocol.OperationOffset+protocol.OperationSize])
	message.ContentLength = binary.BigEndian.Uint64(encodedMessage[protocol.ContentLengthOffset : protocol.ContentLengthOffset+protocol.ContentLengthSize])

	if message.ContentLength > 0 {
		content := make([]byte, message.ContentLength)
		_, err = io.ReadAtLeast(c.conn, content, int(message.ContentLength))
		if err != nil {
			if c.state.Load() != CONNECTED {
				err = c.Error()
				c.logger.Error().Msgf(errors.WithContext(err, POP).Error())
				return nil, nil, errors.WithContext(err, POP)
			}
			c.logger.Error().Msgf(errors.WithContext(err, POP).Error())
			return nil, nil, errors.WithContext(c.closeWithError(err), POP)
		}
		return message, &content, nil
	}

	return message, nil, nil
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

func (c *Sync) pause() error {
	if c.state.CAS(CONNECTED, PAUSED) {
		c.error.Store(ConnectionPaused)
		return nil
	} else if c.state.Load() == PAUSED {
		return ConnectionPaused
	}
	return ConnectionClosed
}

func (c *Sync) close() error {
	if c.state.CAS(CONNECTED, CLOSED) {
		c.error.Store(ConnectionClosed)
		return nil
	} else if c.state.CAS(PAUSED, CLOSED) {
		c.error.Store(ConnectionClosed)
		return nil
	}
	return ConnectionClosed
}

func (c *Sync) closeWithError(err error) error {
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

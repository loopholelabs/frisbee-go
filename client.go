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
	"errors"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"net"
	"sync"
	"time"
)

var (
	ErrNotInitialized        = errors.New("client is not initialized")
	ErrAlreadyInitialized    = errors.New("client is already initialized")
	ErrAlreadyClosed         = errors.New("client is already closed")
	ErrReconnectNotSupported = errors.New("reconnect is not supported when using a pre-existing connection")
	ErrReconnecting          = errors.New("client is reconnecting")
	ErrUnknownState          = errors.New("unknown state")
)

const (
	uninitializedState uint32 = iota
	initializedState
	reconnectingState
	closedState
)

// Client connects to a frisbee Server and can send and receive frisbee packets
type Client struct {
	conn          *Async
	handlerTable  HandlerTable
	ctx           context.Context
	options       *Options
	addr          string
	streamHandler []NewStreamHandler
	state         *atomic.Uint32
	wg            sync.WaitGroup

	// PacketContext is used to define packet-specific contexts based on the incoming packet
	// and is run whenever a new packet arrives
	PacketContext func(context.Context, *packet.Packet) context.Context

	// UpdateContext is used to update a handler-specific context whenever the returned
	// Action from a handler is UPDATE
	UpdateContext func(context.Context, *Async) context.Context
}

// NewClient returns an uninitialized frisbee Client with the registered ClientRouter.
// The ConnectAsync method must then be called to dial the server and initialize the connection.
func NewClient(handlerTable HandlerTable, ctx context.Context, opts ...Option) (*Client, error) {
	for i := uint16(0); i < RESERVED9; i++ {
		if _, ok := handlerTable[i]; ok {
			return nil, InvalidHandlerTable
		}
	}

	options := loadOptions(opts...)
	return &Client{
		handlerTable: handlerTable,
		ctx:          ctx,
		options:      options,
		state:        atomic.NewUint32(uninitializedState),
	}, nil
}

// Connect actually connects to the given frisbee server, and starts the reactor goroutines
// to receive and handle incoming packets. If this function is called, FromConn should not be called.
func (c *Client) Connect(addr string, streamHandler ...NewStreamHandler) error {
	if c.state.CompareAndSwap(uninitializedState, initializedState) {
		c.addr = addr
		c.streamHandler = streamHandler
		c.Logger().Debug().Msgf("connecting to %s", c.addr)
		var frisbeeConn *Async
		var err error
		frisbeeConn, err = ConnectAsync(c.addr, c.options.KeepAlive, c.Logger(), c.options.TLSConfig, c.streamHandler...)
		if err != nil {
			return err
		}
		c.conn = frisbeeConn
		c.Logger().Info().Msgf("connected to %s", c.addr)
		c.wg.Add(1)
		go c.handleConn()
		c.Logger().Debug().Msgf("connection handler started for %s", c.addr)
		return nil
	}
	switch c.state.Load() {
	case reconnectingState:
		return ErrReconnecting
	case closedState:
		return ErrAlreadyClosed
	case initializedState:
		return ErrAlreadyInitialized
	default:
		return ErrUnknownState
	}
}

// FromConn takes a pre-existing connection to a Frisbee server and starts the reactor goroutines
// to receive and handle incoming packets. If this function is called, Connect should not be called.
//
// This function cannot be used with the Reconnect option.
func (c *Client) FromConn(conn net.Conn, streamHandler ...NewStreamHandler) error {
	if c.state.CompareAndSwap(uninitializedState, initializedState) {
		if c.options.Reconnect {
			return ErrReconnectNotSupported
		}
		c.conn = NewAsync(conn, c.Logger(), streamHandler...)
		c.wg.Add(1)
		go c.handleConn()
		c.Logger().Debug().Msgf("Connection handler started for %s", c.conn.RemoteAddr())
		return nil
	}
	switch c.state.Load() {
	case reconnectingState:
		return ErrReconnecting
	case closedState:
		return ErrAlreadyClosed
	case initializedState:
		return ErrAlreadyInitialized
	default:
		return ErrUnknownState
	}
}

// Closed checks whether this client has been closed
func (c *Client) Closed() bool {
	return c.state.Load() == closedState
}

// Reconnecting checks whether this client is currently reconnecting
func (c *Client) Reconnecting() bool {
	return c.state.Load() == reconnectingState
}

// Error checks whether this client has an error
func (c *Client) Error() error {
	if c.state.Load() != uninitializedState {
		return c.conn.Error()
	}
	return ErrNotInitialized
}

// Close closes the frisbee client and kills all the goroutines
func (c *Client) Close() error {
	switch c.state.Swap(closedState) {
	case initializedState:
		err := c.conn.Close()
		if err != nil {
			return err
		}
		c.wg.Wait()
		return nil
	case reconnectingState:
		_ = c.conn.Close()
		c.wg.Wait()
		return nil
	case closedState:
		return ErrAlreadyClosed
	case uninitializedState:
		return ErrNotInitialized
	default:
		return ErrUnknownState
	}
}

// WritePacket sends a frisbee packet.Packet from the client to the server
func (c *Client) WritePacket(p *packet.Packet) (err error) {
	switch c.state.Load() {
	case initializedState:
		err = c.conn.WritePacket(p)
		if err != nil {
			c.Logger().Error().Err(err).Msg("error while writing to frisbee conn")
			if !c.options.Reconnect {
				c.wg.Done()
				_ = c.Close()
			} else {
				c.Logger().Warn().Msg("attempting to reconnect")
				c.startReconnect()
				c.wg.Done()
			}
		}
		return
	case uninitializedState:
		return ErrNotInitialized
	case closedState:
		return ErrAlreadyClosed
	case reconnectingState:
		return ErrReconnecting
	default:
		return ErrUnknownState
	}
}

// Flush flushes any queued frisbee Packets from the client to the server
func (c *Client) Flush() error {
	switch c.state.Load() {
	case initializedState:
		return c.conn.Flush()
	case uninitializedState:
		return ErrNotInitialized
	case closedState:
		return ErrAlreadyClosed
	case reconnectingState:
		return ErrReconnecting
	default:
		return ErrUnknownState
	}
}

// CloseChannel returns a channel that can be listened to see if the underlying connection has been closed
func (c *Client) CloseChannel() (<-chan struct{}, error) {
	switch c.state.Load() {
	case initializedState:
		return c.conn.CloseChannel(), nil
	case uninitializedState:
		return nil, ErrNotInitialized
	case closedState:
		return nil, ErrAlreadyClosed
	case reconnectingState:
		return nil, ErrReconnecting
	default:
		return nil, ErrUnknownState
	}
}

// Raw converts the frisbee client into a normal net.Conn object, and returns it.
// This is especially useful in proxying and streaming scenarios.
func (c *Client) Raw() (net.Conn, error) {
	if c.state.CompareAndSwap(initializedState, closedState) {
		conn := c.conn.Raw()
		c.wg.Wait()
		return conn, nil
	}
	switch c.state.Load() {
	case reconnectingState:
		return nil, ErrReconnecting
	case closedState:
		return nil, ErrAlreadyClosed
	case uninitializedState:
		return nil, ErrNotInitialized
	default:
		return nil, ErrUnknownState
	}
}

// Stream returns a new Stream object that can be used to send and receive frisbee packets
//
// Streams are only valid as long as the underlying connection is valid. If the connection is closed,
// or breaks, then the streams will eventually begin to return errors and must be recreated.
func (c *Client) Stream(id uint16) (*Stream, error) {
	switch c.state.Load() {
	case initializedState:
		return c.conn.NewStream(id), nil
	case uninitializedState:
		return nil, ErrNotInitialized
	case closedState:
		return nil, ErrAlreadyClosed
	case reconnectingState:
		return nil, ErrReconnecting
	default:
		return nil, ErrUnknownState
	}
}

// SetNewStreamHandler sets the callback handler for new streams.
//
// It's important to note that this handler is called for new streams and if it is
// not set then stream packets will be dropped.
//
// It's also important to note that the handler itself is called in its own goroutine to
// avoid blocking the read loop. This means that the handler must be thread-safe.
func (c *Client) SetNewStreamHandler(handler NewStreamHandler) error {
	switch c.state.Load() {
	case initializedState:
		return ErrAlreadyInitialized
	case closedState:
		return ErrAlreadyClosed
	case reconnectingState:
		return ErrReconnecting
	case uninitializedState:
		c.conn.SetNewStreamHandler(handler)
		return nil
	default:
		return ErrUnknownState
	}
}

// Logger returns the client's logger (useful for ClientRouter functions)
func (c *Client) Logger() *zerolog.Logger {
	return c.options.Logger
}

func (c *Client) handleConn() {
	var p *packet.Packet
	var outgoing *packet.Packet
	var action Action
	var err error
	var handlerFunc Handler
	for {
		if c.state.Load() != initializedState {
			c.wg.Done()
			return
		}
		p, err = c.conn.ReadPacket()
		if err != nil {
			c.Logger().Debug().Err(err).Msg("error while getting packet frisbee connection")
			if !c.options.Reconnect {
				c.wg.Done()
				_ = c.Close()
			} else {
				c.Logger().Warn().Msg("attempting to reconnect")
				c.startReconnect()
				c.wg.Done()
			}
			return
		}
		handlerFunc = c.handlerTable[p.Metadata.Operation]
		if handlerFunc != nil {
			packetCtx := c.ctx
			if c.PacketContext != nil {
				packetCtx = c.PacketContext(packetCtx, p)
			}
			outgoing, action = handlerFunc(packetCtx, p)
			if outgoing != nil && outgoing.Metadata.ContentLength == uint32(len(*outgoing.Content)) {
				err = c.conn.WritePacket(outgoing)
				if outgoing != p {
					packet.Put(outgoing)
				}
				packet.Put(p)
				if err != nil {
					c.Logger().Error().Err(err).Msg("error while writing to frisbee conn")
					if !c.options.Reconnect {
						c.wg.Done()
						_ = c.Close()
					} else {
						c.Logger().Warn().Msg("attempting to reconnect")
						c.startReconnect()
						c.wg.Done()
					}
					return
				}
			} else {
				packet.Put(p)
			}
			switch action {
			case NONE:
			case CLOSE:
				c.Logger().Debug().Msgf("Closing connection %s because of CLOSE action", c.conn.RemoteAddr())
				c.wg.Done()
				_ = c.Close()
				return
			}
		} else {
			packet.Put(p)
		}
	}
}

func (c *Client) startReconnect() {
	if c.state.CompareAndSwap(initializedState, reconnectingState) {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			var backoff time.Duration
			for {
				switch c.state.Load() {
				case closedState:
					c.Logger().Warn().Msg("reconnect cancelled because client is closed")
					return
				case initializedState:
					c.Logger().Warn().Msg("reconnect cancelled because client is reconnected")
					return
				case uninitializedState:
					c.Logger().Warn().Msg("reconnect cancelled because client is uninitialized")
					return
				case reconnectingState:
					var err error
					c.conn, err = ConnectAsync(c.addr, c.options.KeepAlive, c.Logger(), c.options.TLSConfig, c.streamHandler...)
					if err == nil {
						c.Logger().Info().Msgf("connected to %s", c.addr)
						c.wg.Add(1)
						go c.handleConn()
						c.Logger().Debug().Msgf("connection handler restarted for %s", c.addr)
						return
					}
					c.Logger().Error().Err(err).Msgf("error while reconnecting to %s", c.addr)
					if backoff == 0 {
						backoff = minBackoff
					} else {
						backoff *= 2
					}
					if backoff > maxBackoff*60*60*24 {
						backoff = maxBackoff * 60 * 60 * 24
					}
					c.Logger().Debug().Msgf("reconnecting, next retry in %s", backoff)
					time.Sleep(backoff)
				default:
					c.Logger().Warn().Msg("reconnect cancelled because client is in unknown state")
				}
			}
		}()
	}
}

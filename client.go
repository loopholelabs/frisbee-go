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
	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"net"
	"sync"
)

// Client connects to a frisbee Server and can send and receive frisbee packets
type Client struct {
	conn             *Async
	handlerTable     HandlerTable
	ctx              context.Context
	options          *Options
	closed           *atomic.Bool
	wg               sync.WaitGroup
	heartbeatChannel chan struct{}

	// PacketContext is used to define packet-specific contexts based on the incomingPackets packet
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
	var heartbeatChannel chan struct{}

	return &Client{
		handlerTable:     handlerTable,
		ctx:              ctx,
		options:          options,
		closed:           atomic.NewBool(false),
		heartbeatChannel: heartbeatChannel,
	}, nil
}

// Connect actually connects to the given frisbee server, and starts the reactor goroutines
// to receive and handle incomingPackets packets. If this function is called, FromConn should not be called.
func (c *Client) Connect(addr string, streamHandler ...NewStreamHandler) error {
	c.Logger().Debug().Msgf("Connecting to %s", addr)
	var frisbeeConn *Async
	var err error
	frisbeeConn, err = ConnectAsync(addr, c.options.KeepAlive, c.Logger(), c.options.TLSConfig, streamHandler...)
	if err != nil {
		return err
	}
	c.conn = frisbeeConn
	c.Logger().Info().Msgf("Connected to %s", addr)

	c.wg.Add(1)
	go c.handleConn()
	c.Logger().Debug().Msgf("Connection handler started for %s", addr)
	return nil
}

// FromConn takes a pre-existing connection to a Frisbee server and starts the reactor goroutines
// to receive and handle incomingPackets packets. If this function is called, Connect should not be called.
func (c *Client) FromConn(conn net.Conn, streamHandler ...NewStreamHandler) error {
	c.conn = NewAsync(conn, c.Logger(), streamHandler...)
	c.wg.Add(1)
	go c.handleConn()
	c.Logger().Debug().Msgf("Connection handler started for %s", c.conn.RemoteAddr())
	return nil
}

// Closed checks whether this client has been closed
func (c *Client) Closed() bool {
	return c.closed.Load()
}

// Error checks whether this client has an error
func (c *Client) Error() error {
	return c.conn.Error()
}

// Close closes the frisbee client and kills all the goroutines
func (c *Client) Close() error {
	if c.closed.CompareAndSwap(false, true) {
		err := c.conn.Close()
		if err != nil {
			return err
		}
		c.wg.Wait()
		return nil
	}
	return c.conn.Close()
}

// WritePacket sends a frisbee packet.Packet from the client to the server
func (c *Client) WritePacket(p *packet.Packet) error {
	return c.conn.WritePacket(p)
}

// Flush flushes any queued frisbee Packets from the client to the server
func (c *Client) Flush() error {
	return c.conn.Flush()
}

// CloseChannel returns a channel that can be listened to see if this client has been closed
func (c *Client) CloseChannel() <-chan struct{} {
	return c.conn.CloseChannel()
}

// Raw converts the frisbee client into a normal net.Conn object, and returns it.
// This is especially useful in proxying and streaming scenarios.
func (c *Client) Raw() (net.Conn, error) {
	if c.conn == nil {
		return nil, ConnectionNotInitialized
	}
	if c.closed.CompareAndSwap(false, true) {
		conn := c.conn.Raw()
		c.wg.Wait()
		return conn, nil
	}
	return c.conn.Raw(), nil
}

// Stream returns a new Stream object that can be used to send and receive frisbee packets
func (c *Client) Stream(id uint16) *Stream {
	return c.conn.NewStream(id)
}

// SetNewStreamHandler sets the callback handler for new streams.
//
// It's important to note that this handler is called for new streams and if it is
// not set then stream packets will be dropped.
//
// It's also important to note that the handler itself is called in its own goroutine to
// avoid blocking the read lop. This means that the handler must be thread-safe.
func (c *Client) SetNewStreamHandler(handler NewStreamHandler) {
	c.conn.SetNewStreamHandler(handler)
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
		if c.closed.Load() {
			c.wg.Done()
			return
		}
		p, err = c.conn.ReadPacket()
		if err != nil {
			c.Logger().Debug().Err(err).Msg("error while getting packet frisbee connection")
			c.wg.Done()
			_ = c.Close()
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
					c.wg.Done()
					_ = c.Close()
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

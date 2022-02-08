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
	"context"
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"net"
	"sync"
	"time"
)

// Client connects to a frisbee Server and can send and receive frisbee messages
type Client struct {
	addr             string
	conn             *Async
	handlerTable     HandlerTable
	ctx              context.Context
	options          *Options
	closed           *atomic.Bool
	wg               sync.WaitGroup
	heartbeatChannel chan struct{}
}

// NewClient returns an uninitialized frisbee Client with the registered ClientRouter.
// The ConnectAsync method must then be called to dial the server and initialize the connection
func NewClient(addr string, handlerTable HandlerTable, ctx context.Context, opts ...Option) (*Client, error) {

	for i := uint16(0); i < RESERVED9; i++ {
		if _, ok := handlerTable[i]; ok {
			return nil, InvalidHandlerTable
		}
	}

	options := loadOptions(opts...)
	var heartbeatChannel chan struct{}
	if options.Heartbeat > time.Duration(0) {
		heartbeatChannel = make(chan struct{}, 1)
		handlerTable[HEARTBEAT] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
			heartbeatChannel <- struct{}{}
			return
		}
	}

	return &Client{
		addr:             addr,
		handlerTable:     handlerTable,
		ctx:              ctx,
		options:          options,
		closed:           atomic.NewBool(false),
		heartbeatChannel: heartbeatChannel,
	}, nil
}

// Connect actually connects to the given frisbee server, and starts the reactor goroutines
// to receive and handle incoming messages.
func (c *Client) Connect() error {
	c.Logger().Debug().Msgf("Connecting to %s", c.addr)
	frisbeeConn, err := ConnectAsync(c.addr, c.options.KeepAlive, c.Logger(), c.options.TLSConfig)
	if err != nil {
		return err
	}
	c.conn = frisbeeConn
	c.Logger().Info().Msgf("Connected to %s", c.addr)

	c.wg.Add(1)
	go c.reactor()
	c.Logger().Debug().Msgf("Reactor started for %s", c.addr)

	if c.options.Heartbeat > time.Duration(0) {
		c.wg.Add(1)
		go c.heartbeat()
		c.Logger().Debug().Msgf("Heartbeat started for %s", c.addr)
	}

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
	if c.closed.CAS(false, true) {
		err := c.conn.Close()
		if err != nil {
			return err
		}
		c.wg.Wait()
		return nil
	}
	return c.conn.Close()
}

// WriteMessage sends a frisbee packet.Packet from the client to the server
func (c *Client) WriteMessage(p *packet.Packet) error {
	return c.conn.WritePacket(p)
}

// Flush flushes any queued frisbee Messages from the client to the server
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
	if c.closed.CAS(false, true) {
		conn := c.conn.Raw()
		c.wg.Wait()
		return conn, nil
	}
	return c.conn.Raw(), nil
}

// Logger returns the client's logger (useful for ClientRouter functions)
func (c *Client) Logger() *zerolog.Logger {
	return c.options.Logger
}

func (c *Client) reactor() {
	for {
		if c.closed.Load() {
			c.wg.Done()
			return
		}
		p, err := c.conn.ReadPacket()
		if err != nil {
			c.Logger().Error().Err(err).Msg("error while reading from frisbee connection")
			c.wg.Done()
			_ = c.Close()
			return
		}

		handlerFunc := c.handlerTable[p.Metadata.Operation]
		if handlerFunc != nil {
			var action Action
			var outgoing *packet.Packet
			outgoing, action = handlerFunc(c.ctx, p)

			if outgoing != nil && outgoing.Metadata.ContentLength == uint32(len(outgoing.Content)) {
				err = c.conn.WritePacket(outgoing)
				if err != nil {
					c.Logger().Error().Err(err).Msg("error while writing to frisbee conn")
					c.wg.Done()
					_ = c.Close()
					return
				}
			}
			if outgoing != p {
				packet.Put(outgoing)
			}
			packet.Put(p)

			switch action {
			case NONE:
			case CLOSE:
				c.Logger().Debug().Msgf("Closing connection %s because of CLOSE action", c.addr)
				c.wg.Done()
				_ = c.Close()
				return
			case SHUTDOWN:
				c.Logger().Debug().Msgf("Closing connection %s because of SHUTDOWN action", c.addr)
				c.wg.Done()
				_ = c.Close()
				return
			default:
			}
		}
	}
}

func (c *Client) heartbeat() {
	for {
		<-time.After(c.options.Heartbeat)
		if c.closed.Load() {
			c.wg.Done()
			return
		}
		if c.conn.WriteBufferSize() == 0 {
			err := c.WriteMessage(HEARTBEATPacket)
			if err != nil {
				c.Logger().Error().Err(err).Msg("error while writing to frisbee conn")
				c.wg.Done()
				_ = c.Close()
				return
			}
			start := time.Now()
			c.Logger().Debug().Msgf("Heartbeat sent at %s", start)
			select {
			case <-c.heartbeatChannel:
				c.Logger().Debug().Msgf("Heartbeat Received with RTT: %d", time.Since(start))
			case <-time.After(c.options.Heartbeat):
				c.Logger().Error().Msg("Heartbeat not received within timeout period")
				c.wg.Done()
				_ = c.Close()
				return
			}
		} else {
			c.Logger().Debug().Msgf("Skipping heartbeat because write buffer size > 0")
		}
	}
}

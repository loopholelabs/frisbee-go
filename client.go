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
	"github.com/loopholelabs/frisbee/internal/errors"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"net"
	"sync"
	"time"
)

// ClientRouterFunc defines a message handler for a specific frisbee message
type ClientRouterFunc func(incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)

// ClientRouter maps frisbee message types to specific handler functions (of type ClientRouterFunc)
type ClientRouter map[uint32]ClientRouterFunc

// Client connects to a frisbee Server and can send and receive frisbee messages
type Client struct {
	addr             string
	conn             *Async
	router           ClientRouter
	options          *Options
	closed           *atomic.Bool
	wg               sync.WaitGroup
	heartbeatChannel chan struct{}
}

// NewClient returns an uninitialized frisbee Client with the registered ClientRouter.
// The ConnectAsync method must then be called to dial the server and initialize the connection
func NewClient(addr string, router ClientRouter, opts ...Option) (*Client, error) {

	for i := uint32(0); i < RESERVED9; i++ {
		if _, ok := router[i]; ok {
			return nil, InvalidRouter
		}
	}

	options := loadOptions(opts...)
	var heartbeatChannel chan struct{}
	if options.Heartbeat > time.Duration(0) {
		heartbeatChannel = make(chan struct{}, 1)
		router[HEARTBEAT] = func(_ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
			heartbeatChannel <- struct{}{}
			return
		}
	}

	return &Client{
		addr:             addr,
		router:           router,
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

// StreamChannel returns a channel that can be listened on to retrieve stream connections as they're created
func (c *Client) StreamChannel() <-chan *Stream {
	return c.conn.StreamChannel()
}

// NewStreamConn creates a new Stream from the underlying frisbee.Async
func (c *Client) NewStreamConn(id uint64) *Stream {
	return c.conn.NewStream(id)
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

// WriteMessage sends a frisbee Message from the client to the server
func (c *Client) WriteMessage(message *Message, content *[]byte) error {
	return c.conn.WriteMessage(message, content)
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
	defer c.wg.Done()
	for {
		if c.closed.Load() {
			return
		}
		incomingMessage, incomingContent, err := c.conn.ReadMessage()
		if err != nil {
			c.Logger().Error().Msgf(errors.WithContext(err, READCONN).Error())
			_ = c.Close()
			return
		}

		routerFunc := c.router[incomingMessage.Operation]
		if routerFunc != nil {
			var outgoingMessage *Message
			var outgoingContent []byte
			var action Action
			if incomingMessage.ContentLength == 0 || incomingContent == nil {
				outgoingMessage, outgoingContent, action = routerFunc(*incomingMessage, nil)
			} else {
				outgoingMessage, outgoingContent, action = routerFunc(*incomingMessage, *incomingContent)
			}

			if outgoingMessage != nil && outgoingMessage.ContentLength == uint64(len(outgoingContent)) {
				err = c.conn.WriteMessage(outgoingMessage, &outgoingContent)
				if err != nil {
					c.Logger().Error().Msgf(errors.WithContext(err, WRITECONN).Error())
					_ = c.Close()
					return
				}
			}

			switch action {
			case CLOSE:
				c.Logger().Debug().Msgf("Closing connection %s because of CLOSE action", c.addr)
				_ = c.Close()
				return
			case SHUTDOWN:
				c.Logger().Debug().Msgf("Closing connection %s because of SHUTDOWN action", c.addr)
				_ = c.Close()
				return
			default:
			}
		}
	}
}

func (c *Client) heartbeat() {
	defer c.wg.Done()
	for {
		<-time.After(c.options.Heartbeat)
		if c.closed.Load() {
			return
		}
		if c.conn.WriteBufferSize() == 0 {
			err := c.WriteMessage(&Message{
				Operation: HEARTBEAT,
			}, nil)
			if err != nil {
				c.Logger().Error().Msgf(errors.WithContext(err, WRITECONN).Error())
				_ = c.Close()
				return
			}
			start := time.Now()
			c.Logger().Debug().Msgf("Heartbeat sent at %s", start)
			<-c.heartbeatChannel
			c.Logger().Debug().Msgf("Heartbeat Received with RTT: %d", time.Since(start))
		} else {
			c.Logger().Debug().Msgf("Skipping heartbeat because write buffer size > 0")
		}
	}
}

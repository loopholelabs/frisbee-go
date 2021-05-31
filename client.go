package frisbee

import (
	"github.com/loophole-labs/frisbee/internal/errors"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"net"
	"time"
)

// ClientRouterFunc defines a message handler type
type ClientRouterFunc func(incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)

// ClientRouter defines map of message handlers
type ClientRouter map[uint32]ClientRouterFunc

// Client accepts and handles inbound messages
type Client struct {
	addr             string
	conn             *Conn
	router           ClientRouter
	options          *Options
	messageOffset    uint32
	closed           *atomic.Bool
	heartbeatChannel chan struct{}
}

// NewClient returns an initialized client
func NewClient(addr string, router ClientRouter, opts ...Option) *Client {

	options := loadOptions(opts...)
	messageOffset := uint32(0)
	newRouter := ClientRouter{}

	heartbeatChannel := make(chan struct{}, 1)

	if options.Heartbeat != time.Duration(-1) {
		newRouter[messageOffset] = func(_ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
			heartbeatChannel <- struct{}{}
			return
		}
		messageOffset++
	}

	for message, handler := range router {
		newRouter[message+messageOffset] = handler
	}

	options.Logger.Printf("Message offset: %d", messageOffset)

	return &Client{
		addr:             addr,
		router:           newRouter,
		options:          options,
		messageOffset:    messageOffset,
		closed:           atomic.NewBool(false),
		heartbeatChannel: heartbeatChannel,
	}
}

func (c *Client) Connect() error {
	c.Logger().Debug().Msgf("Connecting to %s", c.addr)
	frisbeeConn, err := Connect("tcp", c.addr, c.options.KeepAlive, c.Logger(), c.messageOffset)
	if err != nil {
		return err
	}
	c.conn = frisbeeConn
	c.Logger().Info().Msgf("Connected to %s", c.addr)

	go c.reactor()
	c.Logger().Debug().Msgf("Reactor started for %s", c.addr)

	if c.options.Heartbeat != time.Duration(-1) {
		go c.heartbeat()
		c.Logger().Debug().Msgf("Heartbeat started for %s", c.addr)
	}

	return nil
}

func (c *Client) Close() error {
	c.closed.Store(true)
	return c.conn.Close()
}

func (c *Client) Write(message *Message, content *[]byte) error {
	return c.conn.Write(message, content)
}

func (c *Client) Raw() (net.Conn, error) {
	if c.conn == nil {
		return nil, ConnectionNotInitialized
	}
	c.closed.Store(true)
	return c.conn.Raw(), nil
}

func (c *Client) Logger() *zerolog.Logger {
	return c.options.Logger
}

func (c *Client) reactor() {
	for {
		incomingMessage, incomingContent, err := c.conn.Read()
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
				err = c.conn.Write(outgoingMessage, &outgoingContent)
				if err != nil {
					c.Logger().Error().Msgf(errors.WithContext(err, WRITECONN).Error())
					_ = c.Close()
					return
				}
			}

			switch action {
			case Close:
				c.Logger().Debug().Msgf("Closing connection %s because of CLOSE action", c.addr)
				_ = c.Close()
				return
			case Shutdown:
				c.Logger().Debug().Msgf("Closing connection %s because of SHUTDOWN action", c.addr)
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
		if c.conn.WriteBufferSize() == 0 {
			err := c.Write(&Message{
				Operation: HEARTBEAT - c.messageOffset,
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

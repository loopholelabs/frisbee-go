package frisbee

import (
	"github.com/loophole-labs/frisbee/internal/errors"
	"github.com/rs/zerolog"
	"net"
)

// ClientRouterFunc defines a message handler type
type ClientRouterFunc func(incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)

// ClientRouter defines map of message handlers
type ClientRouter map[uint32]ClientRouterFunc

// Client accepts and handles inbound messages
type Client struct {
	addr          string
	conn          *Conn
	router        ClientRouter
	options       *Options
	messageOffset uint32
	closed        bool
}

// NewClient returns an initialized client
func NewClient(addr string, router ClientRouter, opts ...Option) *Client {

	options := loadOptions(opts...)
	messageOffset := uint32(0)
	newRouter := ClientRouter{}

	if options.Heartbeat {
		newRouter[messageOffset] = handleHeartbeat
		messageOffset++
	}

	for message, handler := range router {
		newRouter[message+messageOffset] = handler
	}

	return &Client{
		addr:          addr,
		router:        newRouter,
		options:       options,
		messageOffset: messageOffset,
		closed:        false,
	}
}

func (c *Client) Connect() error {
	c.Logger().Debug().Msgf("Connecting to %s", c.addr)
	frisbeeConn, err := Connect("tcp", c.addr, c.options.KeepAlive, c.Logger())
	if err != nil {
		return err
	}
	c.conn = frisbeeConn
	c.Logger().Info().Msgf("Connected to %s", c.addr)

	go c.reactor()

	c.Logger().Debug().Msgf("Reactor started for %s", c.addr)

	return nil
}

func (c *Client) Close() error {
	c.closed = true
	return c.conn.Close()
}

func (c *Client) Write(message *Message, content *[]byte) error {
	return c.conn.Write(message, content)
}

func (c *Client) Raw() (net.Conn, error) {
	if c.conn == nil {
		return nil, ConnectionNotInitialized
	}
	c.closed = true
	return c.conn.Raw(), nil
}

func (c *Client) Logger() *zerolog.Logger {
	return c.options.Logger
}

func (c *Client) reactor() {
	for {
		incomingMessage, incomingContent, err := c.conn.Read()
		if err != nil {
			c.Logger().Error().Msgf(errors.WithContext(err, CLOSE).Error())
			_ = c.Close()
			return
		}

		routerFunc := c.router[incomingMessage.Operation+c.messageOffset]
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
				err := c.conn.Write(outgoingMessage, &outgoingContent)
				if err != nil {
					c.Logger().Error().Msgf(errors.WithContext(err, CLOSE).Error())
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

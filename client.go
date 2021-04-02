package frisbee

import "github.com/rs/zerolog"

type ClientRouterFunc func(incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)
type ClientRouter map[uint16]ClientRouterFunc

type Client struct {
	addr    string
	Conn    *Conn
	router  ClientRouter
	Options *Options
	closed  bool
}

func NewClient(addr string, router ClientRouter, opts ...Option) *Client {
	return &Client{
		addr:    addr,
		router:  router,
		Options: LoadOptions(opts...),
		closed:  false,
	}
}

func (c *Client) Connect() error {
	c.Options.Logger.Debug().Msgf("Connecting to %s", c.addr)
	frisbeeConn, err := Connect("tcp", c.addr, c.Options.KeepAlive, nil)
	if err != nil {
		return err
	}
	c.Conn = frisbeeConn
	c.logger().Info().Msgf("Connected to %s", c.addr)

	// Reacts to incoming messages
	go c.reactor()

	c.logger().Debug().Msg("Reactor started")

	return nil
}

func (c *Client) logger() *zerolog.Logger {
	return c.Options.Logger
}

func (c *Client) Close() error {
	c.closed = true
	return c.Conn.Close()
}

func (c *Client) Write(message *Message, content *[]byte) error {
	return c.Conn.Write(message, content)
}

func (c *Client) reactor() {
	for {
		incomingMessage, incomingContent, err := c.Conn.Read()
		if err != nil {
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

			if outgoingMessage != nil && outgoingMessage.ContentLength == uint32(len(outgoingContent)) {
				err := c.Conn.Write(outgoingMessage, &outgoingContent)
				if err != nil {
					_ = c.Close()
					return
				}
			}

			switch action {
			case Close:
				_ = c.Close()
				return
			case Shutdown:
				_ = c.Close()
				return
			default:
			}
		}
	}
}

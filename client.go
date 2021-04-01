package frisbee

type ClientRouterFunc func(incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)
type ClientRouter map[uint16]ClientRouterFunc

type Client struct {
	addr    string
	Conn    *Conn
	router  ClientRouter
	options *Options
	closed  bool
}

func NewClient(addr string, router ClientRouter, opts ...Option) *Client {
	return &Client{
		addr:    addr,
		router:  router,
		options: LoadOptions(opts...),
		closed:  false,
	}
}

func (c *Client) Connect() error {
	c.options.Logger.Debug().Msgf("Connecting to %s", c.addr)
	frisbeeConn, err := Connect("tcp", c.addr, c.options.KeepAlive)
	if err != nil {
		return err
	}
	c.Conn = frisbeeConn
	c.options.Logger.Info().Msgf("Connected to %s", c.addr)

	// Reacts to incoming messages
	go c.reactor()

	c.options.Logger.Debug().Msg("Reactor started")

	return nil
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
				outgoingMessage, outgoingContent, action = routerFunc(Message(*incomingMessage), nil)
			} else {
				outgoingMessage, outgoingContent, action = routerFunc(Message(*incomingMessage), *incomingContent)
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

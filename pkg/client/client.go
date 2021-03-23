package client

import (
	"bufio"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/codec"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/pkg/errors"
	"net"
	"sync"
)

type Client struct {
	addr          string
	Conn          *net.TCPConn
	bufConnReader *bufio.Reader
	bufConnWrite  *bufio.Writer
	packets       sync.Map
	messages      chan uint32
	router        frisbee.ClientRouter
	options       *Options
	writer        chan []byte
	quit          chan struct{}
}

func NewClient(addr string, router frisbee.ClientRouter, opts ...Option) *Client {
	return &Client{
		addr:     addr,
		router:   router,
		options:  LoadOptions(opts...),
		messages: make(chan uint32, 1<<18),
		writer:   make(chan []byte, 1<<18),
		quit:     make(chan struct{}),
	}
}

func (c *Client) Connect() (err error) {
	c.options.Logger.Debug().Msgf("Connecting to %s", c.addr)
	conn, err := net.Dial("tcp4", c.addr)
	c.Conn = conn.(*net.TCPConn)
	_ = c.Conn.SetNoDelay(true)
	_ = c.Conn.SetKeepAlive(true)
	_ = c.Conn.SetKeepAlivePeriod(c.options.KeepAlive)
	c.bufConnReader = bufio.NewReaderSize(c.Conn, 1<<18)
	c.bufConnWrite = bufio.NewWriterSize(c.Conn, 1<<18)
	if err != nil {
		panic(err)
	}
	c.options.Logger.Info().Msgf("Connected to %s", c.addr)
	// Writes to the bufConnWrite
	go Writer(&c.quit, c.bufConnWrite, &c.writer)

	// Reads data from the ringBuffer
	go Decoder(&c.quit, &c.packets, &c.messages, c.bufConnReader)

	// Reacts to incoming messages
	go Reactor(c)

	c.options.Logger.Debug().Msg("Event loops started")

	return
}

func (c *Client) Stop() error {
	close(c.quit)
	return c.Conn.Close()
}

func (c *Client) Write(message frisbee.Message, content *[]byte) error {
	if int(message.ContentLength) != len(*content) {
		return errors.New("invalid content length")
	}

	encodedMessage, err := protocol.EncodeV0(message.Id, message.Operation, message.Routing, message.ContentLength)
	if err != nil {
		return err
	}
	c.options.Logger.Debug().Msgf("Queuing message %d", message.Id)
	c.writer <- encodedMessage[:]
	c.writer <- *content
	return nil
}

func Reactor(c *Client) {
	for {
		select {
		case <-c.quit:
			return
		case id := <-c.messages:
			packetInterface, _ := c.packets.Load(id)
			packet := packetInterface.(*codec.Packet)
			c.options.Logger.Debug().Msgf("Received message %d", packet.Message.Id)
			handlerFunc := c.router[packet.Message.Operation]
			if handlerFunc != nil {
				message, output, action := handlerFunc(frisbee.Message(*packet.Message), packet.Content)

				if message != nil && message.ContentLength == uint32(len(output)) {
					_ = c.Write(*message, &output)
				}
				switch action {
				case frisbee.Close:
					_ = c.Stop()
				case frisbee.Shutdown:
					_ = c.Stop()
				default:
				}
			}
		}
	}
}

func Writer(quit *chan struct{}, bufConnWriter *bufio.Writer, writer *chan []byte) {
	for {
		select {
		case <-*quit:
			_ = bufConnWriter.Flush()
			return
		default:
		}
		for data := range *writer {
			dataLen := len(data)
			written := 0
		LoopedWrite:
			n, _ := bufConnWriter.Write((data)[written:])
			written += n
			if written != dataLen {
				goto LoopedWrite
			}
			if len(*writer) < 1 {
				_ = bufConnWriter.Flush()
				break
			}
		}
	}
}

func Decoder(quit *chan struct{}, packets *sync.Map, messages *chan uint32, bufConnReader *bufio.Reader) {
	for {
		select {
		case <-*quit:
			return
		default:
			if message, _ := bufConnReader.Peek(protocol.HeaderLengthV0); len(message) == protocol.HeaderLengthV0 {
				decodedMessage, err := protocol.DecodeV0(message)
				if err != nil {
					_, _ = bufConnReader.Discard(bufConnReader.Buffered())
					continue
				}
				packet := &codec.Packet{
					Message: &decodedMessage,
				}
				if decodedMessage.ContentLength > 0 {
					if content, _ := bufConnReader.Peek(int(decodedMessage.ContentLength + protocol.HeaderLengthV0)); len(content) == int(decodedMessage.ContentLength+protocol.HeaderLengthV0) {
						_, _ = bufConnReader.Discard(int(decodedMessage.ContentLength + protocol.HeaderLengthV0))
						packet.Content = content[protocol.HeaderLengthV0:]
						(*packets).Store(decodedMessage.Id, packet)
						*messages <- decodedMessage.Id
						continue
					}
					continue
				}
				_, _ = bufConnReader.Discard(protocol.HeaderLengthV0)
				(*packets).Store(decodedMessage.Id, packet)
				*messages <- decodedMessage.Id
				continue
			}
		}
	}
}

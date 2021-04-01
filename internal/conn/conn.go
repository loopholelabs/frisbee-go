package conn

import (
	"bufio"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/pkg/errors"
	"net"
	"sync"
)

const (
	defaultSize = 1 << 18
)

type packet struct {
	message *protocol.MessageV0
	content *[]byte
}

type Conn struct {
	sync.Mutex
	conn     net.Conn
	writer   *bufio.Writer
	flush    chan struct{}
	reader   *bufio.Reader
	context  interface{}
	messages chan *packet
	closed   bool
}

func New(c net.Conn) (conn *Conn) {
	conn = &Conn{
		conn:     c,
		writer:   bufio.NewWriterSize(c, defaultSize),
		flush:    make(chan struct{}, 1024),
		reader:   bufio.NewReaderSize(c, defaultSize),
		messages: make(chan *packet, defaultSize),
		closed:   false,
		context:  nil,
	}

	go conn.flushLoop()
	go conn.readLoop()

	return
}

func (c *Conn) Context() interface{} {
	return c.context
}

func (c *Conn) SetContext(ctx interface{}) {
	c.context = ctx
}

func (c *Conn) LocalAddr() net.Addr {
	return c.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.RemoteAddr()
}

func (c *Conn) flushLoop() {
	for {
		if _, ok := <-c.flush; !ok {
			return
		}
		c.Lock()
		if c.writer.Buffered() > 0 {
			_ = c.writer.Flush()
		}
		c.Unlock()
	}
}

func (c *Conn) Write(message *protocol.MessageV0, content *[]byte) error {
	if content != nil && int(message.ContentLength) != len(*content) {
		return errors.New("invalid content length")
	}

	if c.closed {
		return errors.New("connection closed")
	}

	encodedMessage, err := protocol.EncodeV0(message.Id, message.Operation, message.Routing, message.ContentLength)
	if err != nil {
		return err
	}
	c.Lock()
	_, _ = c.writer.Write(encodedMessage[:])
	if content != nil {
		_, _ = c.writer.Write(*content)
	}

	if len(c.flush) == 0 {
		select {
		case c.flush <- struct{}{}:
		default:
		}
	}

	// TODO: benchmark this before the len(c.flush) if statement
	c.Unlock()

	return nil
}

func (c *Conn) readLoop() {
	for {
		message, err := c.reader.Peek(protocol.HeaderLengthV0)
		if len(message) == protocol.HeaderLengthV0 {
			decodedMessage, err := protocol.DecodeV0(message)
			if err != nil {
				_, _ = c.reader.Discard(c.reader.Buffered())
				continue
			}
			if decodedMessage.ContentLength > 0 {
				if content, _ := c.reader.Peek(int(decodedMessage.ContentLength + protocol.HeaderLengthV0)); len(content) == int(decodedMessage.ContentLength+protocol.HeaderLengthV0) {
					readContent := make([]byte, decodedMessage.ContentLength)
					copy(readContent, content[protocol.HeaderLengthV0:])
					_, _ = c.reader.Discard(int(decodedMessage.ContentLength + protocol.HeaderLengthV0))
					readPacket := &packet{
						message: &decodedMessage,
						content: &readContent,
					}
					c.messages <- readPacket
				}
				continue
			}
			_, _ = c.reader.Discard(protocol.HeaderLengthV0)
			readPacket := &packet{
				message: &decodedMessage,
				content: nil,
			}
			c.messages <- readPacket
		}
		if err != nil {
			_ = c.Close()
			return
		}
		// TODO: Check if running Gosched here adds performance
		//runtime.Gosched()
	}
}

func (c *Conn) Read() (*protocol.MessageV0, *[]byte, error) {

	if c.closed {
		return nil, nil, errors.New("connection closed")
	}
	readPacket := <-c.messages
	return readPacket.message, readPacket.content, nil
}

func (c *Conn) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	c.Lock()
	defer c.Unlock()
	close(c.flush)
	close(c.messages)
	c.closed = true
	return c.conn.Close()
}

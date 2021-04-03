package frisbee

import (
	"bufio"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/loophole-labs/frisbee/internal/ringbuffer"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

var silentLogger = zerolog.New(os.Stdout)

type Conn struct {
	sync.Mutex
	conn     net.Conn
	writer   *bufio.Writer
	flush    *sync.Cond
	reader   *bufio.Reader
	context  interface{}
	messages *ringbuffer.RingBuffer
	closed   bool
	logger   *zerolog.Logger
	Error    error
}

func Connect(network string, addr string, keepAlive time.Duration, l *zerolog.Logger) (*Conn, error) {
	conn, err := net.Dial(network, addr)
	_ = conn.(*net.TCPConn).SetKeepAlive(true)
	_ = conn.(*net.TCPConn).SetKeepAlivePeriod(keepAlive)
	if err != nil {
		return nil, err
	}
	return New(conn, l), nil
}

func New(c net.Conn, l *zerolog.Logger) (conn *Conn) {
	conn = &Conn{
		conn:     c,
		writer:   bufio.NewWriterSize(c, 1<<18),
		reader:   bufio.NewReaderSize(c, 1<<22),
		messages: ringbuffer.NewRingBuffer(1 << 18),
		closed:   false,
		context:  nil,
		logger:   l,
	}

	if l == nil {
		conn.logger = &silentLogger
	}

	conn.flush = sync.NewCond(conn)

	go conn.flushLoop()
	go conn.readLoop()

	return
}

func (c *Conn) Raw() net.Conn {
	c.Lock()
	defer c.Unlock()
	c.messages.Close()
	c.closed = true
	c.flush.Broadcast()
	return c.conn
}

func (c *Conn) Context() interface{} {
	return c.context
}

func (c *Conn) SetContext(ctx interface{}) {
	c.context = ctx
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) flushLoop() {
	for {
		c.flush.L.Lock()
		for c.writer.Buffered() == 0 {
			c.flush.Wait()
		}
		if c.closed {
			c.flush.L.Unlock()
			return
		}
		err := c.writer.Flush()
		if err != nil {
			_ = c.close(err)
		}
		c.flush.L.Unlock()
	}
}

func (c *Conn) Write(message *Message, content *[]byte) error {
	if content != nil && int(message.ContentLength) != len(*content) {
		return errors.New("invalid content length")
	}

	if c.closed {
		return errors.New("connection closed")
	}
	encodedMessage, _ := protocol.EncodeV0(message.Id, message.Operation, message.Routing, message.ContentLength)
	c.Lock()
	_, err := c.writer.Write(encodedMessage[:])
	if err != nil {
		c.logger.Error().Msgf("Short write to write buffer %+v", err)
	}
	if content != nil {
		_, err = c.writer.Write(*content)
		if err != nil {
			c.logger.Error().Msgf("Short write to write buffer %+v", err)
		}
	}
	c.Unlock()

	c.flush.Signal()

	return nil
}

func (c *Conn) readLoop() {
	for {
		message, err := c.reader.Peek(protocol.HeaderLengthV0)
		if err != nil {
			_ = c.close(err)
			return
		}
		if len(message) == protocol.HeaderLengthV0 {
			decodedMessage, err := protocol.DecodeV0(message)
			if err != nil {
				c.logger.Error().Msgf("Invalid read buffer contents, discarding")
				_, err = c.reader.Discard(c.reader.Buffered())
				if err != nil {
					c.logger.Debug().Msgf("Error while discarding from read buffer: %+v", err)
				}
				continue
			}
			if decodedMessage.ContentLength > 0 {
				content, err := c.reader.Peek(int(decodedMessage.ContentLength + protocol.HeaderLengthV0))
				if err != nil {
					_ = c.close(err)
					return
				}
				if len(content) == int(decodedMessage.ContentLength+protocol.HeaderLengthV0) {
					readContent := make([]byte, decodedMessage.ContentLength)
					copy(readContent, content[protocol.HeaderLengthV0:])
					_, err = c.reader.Discard(int(decodedMessage.ContentLength + protocol.HeaderLengthV0))
					if err != nil {
						c.logger.Debug().Msgf("Error while discarding from read buffer: %+v", err)
					}
					readPacket := &protocol.PacketV0{
						Message: &decodedMessage,
						Content: &readContent,
					}
					err = c.messages.Push(readPacket)
					if err != nil {
						c.logger.Debug().Msgf("Error pushing read packet to message queue %+v", err)
					}
				}
				continue
			}
			_, err = c.reader.Discard(protocol.HeaderLengthV0)
			if err != nil {
				c.logger.Debug().Msgf("Error while discarding from read buffer: %+v", err)
			}
			readPacket := &protocol.PacketV0{
				Message: &decodedMessage,
				Content: nil,
			}
			err = c.messages.Push(readPacket)
			if err != nil {
				c.logger.Debug().Msgf("Error pushing read packet to message queue %+v", err)
			}
		}
	}
}

func (c *Conn) Read() (*Message, *[]byte, error) {
	if c.closed {
		return nil, nil, errors.New("connection closed")
	}

	readPacket, err := c.messages.Pop()
	if err != nil {
		return nil, nil, errors.New("unable to retrieve packet")
	}

	return (*Message)(readPacket.Message), readPacket.Content, nil
}

func (c *Conn) Logger() *zerolog.Logger {
	return c.logger
}

func (c *Conn) close(connError error) (err error) {
	if c.closed {
		return c.Error
	}
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	conn := c.Raw()
	if connError != nil && connError != io.EOF {
		c.logger.Error().Msgf("Closing connection with error %+v", connError)
		c.Error = connError
		closeError := conn.Close()
		if closeError != nil {
			c.logger.Error().Msgf("Error while closing connection %+v", closeError)
		}
		return c.Error
	}
	c.Error = conn.Close()
	return c.Error
}

func (c *Conn) Close() (err error) {
	return c.close(nil)
}

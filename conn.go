package frisbee

import (
	"bufio"
	"encoding/binary"
	"github.com/gobwas/pool/pbufio"
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

var (
	writePool    = pbufio.NewWriterPool(1024, 1<<32)
	silentLogger = zerolog.New(os.Stdout)
)

type Conn struct {
	sync.Mutex
	conn     net.Conn
	writer   *bufio.Writer
	flush    *sync.Cond
	context  interface{}
	messages *ringbuffer.RingBuffer
	closed   bool
	logger   *zerolog.Logger
	Error    error
	wg       sync.WaitGroup
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
		writer:   writePool.Get(c, 1<<19),
		messages: ringbuffer.NewRingBuffer(1 << 19),
		closed:   false,
		context:  nil,
		logger:   l,
	}

	if l == nil {
		conn.logger = &silentLogger
	}

	conn.flush = sync.NewCond(conn)

	conn.wg.Add(2)
	go conn.flushLoop()
	go conn.readLoop()

	return
}

func (c *Conn) Raw() net.Conn {
	c.Lock()
	c.messages.Close()
	c.closed = true
	c.flush.Broadcast()
	writePool.Put(c.writer)
	_ = c.conn.SetReadDeadline(time.Now())
	c.Unlock()
	c.wg.Wait()
	_ = c.conn.SetReadDeadline(time.Time{})
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
	defer c.wg.Done()
	for {
		c.flush.L.Lock()
		for c.writer.Buffered() == 0 {
			if c.closed {
				break
			}
			c.flush.Wait()
		}
		if c.closed {
			c.flush.L.Unlock()
			break
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

	var encodedMessage [protocol.HeaderLengthV0]byte
	encodedMessage[1] = protocol.Version0
	binary.BigEndian.PutUint32(encodedMessage[2:6], message.Id)
	binary.BigEndian.PutUint16(encodedMessage[6:8], message.Operation)
	binary.BigEndian.PutUint32(encodedMessage[8:12], message.Routing)
	binary.BigEndian.PutUint32(encodedMessage[12:16], message.ContentLength)
	c.Lock()
	_, err := c.writer.Write(encodedMessage[:])
	if err != nil {
		c.logger.Error().Msgf("short write to write buffer %+v", err)
	}
	if content != nil {
		_, err = c.writer.Write(*content)
		if err != nil {
			c.logger.Error().Msgf("short write to write buffer %+v", err)
		}
	}
	c.Unlock()

	c.flush.Signal()

	return nil
}

func (c *Conn) readAtLeast(buf []byte, min int) (n int, err error) {
	if len(buf) < min {
		return 0, errors.New("length of buffer too small")
	}
	for n < min && err == nil {
		var nn int
		nn, err = c.conn.Read(buf[n:])
		n += nn
	}
	if n >= min {
		err = nil
	} else if n > 0 && err == io.EOF {
		err = errors.New("unexpected EOF")
	}
	return
}

func (c *Conn) readLoop() {
	defer c.wg.Done()
	buf := make([]byte, 1<<19)
	var index int
	for {
		n, err := c.readAtLeast(buf[:cap(buf)], protocol.HeaderLengthV0)
		if err != nil {
			if !os.IsTimeout(err) {
				_ = c.close(err)
			}
			break
		}

		index = 0
		for index < n {
			if buf[index+1] != protocol.Version0 {
				c.Logger().Error().Msgf("invalid buf contents, discarding")
				break
			}

			decodedMessage := protocol.MessageV0{
				Id:            binary.BigEndian.Uint32(buf[index+2 : index+6]),
				Operation:     binary.BigEndian.Uint16(buf[index+6 : index+8]),
				Routing:       binary.BigEndian.Uint32(buf[index+8 : index+12]),
				ContentLength: binary.BigEndian.Uint32(buf[index+12 : index+16]),
			}
			index += protocol.HeaderLengthV0
			if decodedMessage.ContentLength > 0 {
				readContent := make([]byte, decodedMessage.ContentLength)
				if n-index < int(decodedMessage.ContentLength) {
					for cap(buf) < int(decodedMessage.ContentLength) {
						buf = append(buf[:cap(buf)], 0)
						buf = buf[:cap(buf)]
					}
					cp := copy(readContent, buf[index:n])
					n, err = c.readAtLeast(buf[:cap(buf)], int(decodedMessage.ContentLength)-cp)
					if err != nil {
						if !os.IsTimeout(err) {
							_ = c.close(err)
						}
						return
					}
					copy(readContent[cp:], buf[:int(decodedMessage.ContentLength)-cp])
					index = int(decodedMessage.ContentLength) - cp
				} else {
					copy(readContent, buf[index:index+int(decodedMessage.ContentLength)])
					index += int(decodedMessage.ContentLength)
				}
				err = c.messages.Push(&protocol.PacketV0{
					Message: &decodedMessage,
					Content: &readContent,
				})
				if err != nil {
					c.Logger().Debug().Msgf("Error pushing read packet to message queue %+v", err)
				}
			} else {
				err = c.messages.Push(&protocol.PacketV0{
					Message: &decodedMessage,
					Content: nil,
				})
				if err != nil {
					c.Logger().Debug().Msgf("Error pushing read packet to message queue %+v", err)
				}
			}
			if n == index {
				index = 0
				n, err = c.readAtLeast(buf[:cap(buf)], protocol.HeaderLengthV0)
				if err != nil {
					if !os.IsTimeout(err) {
						_ = c.close(err)
					}
					return
				}
			} else if n-index < protocol.HeaderLengthV0 {
				copy(buf, buf[index:n])
				n -= index
				index = n
				n, err = c.readAtLeast(buf[n:cap(buf)], protocol.HeaderLengthV0-index)
				if err != nil {
					if !os.IsTimeout(err) {
						_ = c.close(err)
					}
					return
				}
				n += index
				index = 0
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
		if closeError != nil && closeError != io.EOF {
			c.logger.Error().Msgf("Error while closing connection %+v", closeError)
		}
		return c.Error
	}
	c.Error = conn.Close()
	return c.Error
}

func (c *Conn) Close() error {
	return c.close(nil)
}

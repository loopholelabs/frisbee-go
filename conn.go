package frisbee

import (
	"bufio"
	"encoding/binary"
	"github.com/gobwas/pool/pbufio"
	"github.com/loophole-labs/frisbee/internal/errors"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/loophole-labs/frisbee/internal/ringbuffer"
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
	flush    chan struct{}
	messages *ringbuffer.RingBuffer
	closed   bool
	logger   *zerolog.Logger
	Error    error
	wg       sync.WaitGroup
}

func Connect(network string, addr string, keepAlive time.Duration, l *zerolog.Logger) (*Conn, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, errors.WithContext(err, DIAL)
	}
	_ = conn.(*net.TCPConn).SetKeepAlive(true)
	_ = conn.(*net.TCPConn).SetKeepAlivePeriod(keepAlive)

	return New(conn, l), nil
}

func New(c net.Conn, l *zerolog.Logger) (conn *Conn) {
	conn = &Conn{
		conn:     c,
		writer:   writePool.Get(c, 1<<19),
		messages: ringbuffer.NewRingBuffer(1 << 19),
		closed:   false,
		flush:    make(chan struct{}, 1024),
		logger:   l,
	}

	if l == nil {
		conn.logger = &silentLogger
	}

	conn.wg.Add(2)
	go conn.flushLoop()
	go conn.readLoop()

	return
}

func (c *Conn) Raw() net.Conn {
	if c.closed {
		return c.conn
	}
	c.Lock()
	c.messages.Close()
	c.closed = true
	close(c.flush)
	if c.writer.Buffered() > 0 {
		_ = c.writer.Flush()
	}
	writePool.Put(c.writer)
	_ = c.conn.SetReadDeadline(time.Now())
	c.Unlock()
	c.wg.Wait()
	_ = c.conn.SetReadDeadline(time.Time{})
	return c.conn
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
		if _, ok := <-c.flush; !ok {
			return
		}
		c.Lock()
		if c.writer.Buffered() > 0 {
			err := c.writer.Flush()
			if err != nil {
				_ = c.close(err)
			}
		}
		c.Unlock()
	}
}

func (c *Conn) Write(message *Message, content *[]byte) error {
	if content != nil && int(message.ContentLength) != len(*content) {
		return InvalidContentLength
	}

	if c.closed {
		return ConnectionClosed
	}

	var encodedMessage [protocol.MessageV0Size]byte

	binary.BigEndian.PutUint16(encodedMessage[protocol.VersionV0Offset:protocol.VersionV0Offset+protocol.VersionV0Size], protocol.Version0)
	binary.BigEndian.PutUint32(encodedMessage[protocol.FromV0Offset:protocol.FromV0Offset+protocol.FromV0Size], message.From)
	binary.BigEndian.PutUint32(encodedMessage[protocol.ToV0Offset:protocol.ToV0Offset+protocol.ToV0Size], message.To)
	binary.BigEndian.PutUint32(encodedMessage[protocol.IdV0Offset:protocol.IdV0Offset+protocol.IdV0Size], message.Id)
	binary.BigEndian.PutUint32(encodedMessage[protocol.OperationV0Offset:protocol.OperationV0Offset+protocol.OperationV0Size], message.Operation)
	binary.BigEndian.PutUint64(encodedMessage[protocol.ContentLengthV0Offset:protocol.ContentLengthV0Offset+protocol.ContentLengthV0Size], message.ContentLength)

	c.Lock()
	_, err := c.writer.Write(encodedMessage[:])
	if err != nil {
		c.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
	}
	if content != nil {
		_, err = c.writer.Write(*content)
		if err != nil {
			c.logger.Error().Msgf(errors.WithContext(err, WRITE).Error())
		}
	}
	c.Unlock()

	if len(c.flush) == 0 {
		select {
		case c.flush <- struct{}{}:
		default:
		}
	}

	return nil
}

func (c *Conn) readAtLeast(buf []byte, min int) (n int, err error) {
	if len(buf) < min {
		return 0, InvalidBufferLength
	}
	for n < min && err == nil {
		var nn int
		nn, err = c.conn.Read(buf[n:])
		n += nn
	}
	if n >= min {
		err = nil
	}
	return
}

func (c *Conn) readLoop() {
	defer c.wg.Done()
	buf := make([]byte, 1<<19)
	var index int
	for {
		n, err := c.readAtLeast(buf[:cap(buf)], protocol.MessageV0Size)
		if err != nil {
			if !os.IsTimeout(err) {
				_ = c.close(err)
			}
			break
		}

		index = 0
		for index < n {
			if binary.BigEndian.Uint16(buf[index+protocol.VersionV0Offset:index+protocol.VersionV0Offset+protocol.VersionV0Size]) != protocol.Version0 {
				c.Logger().Error().Msgf(InvalidBufferContents.Error())
				break
			}

			decodedMessage := protocol.MessageV0{
				From:          binary.BigEndian.Uint32(buf[index+protocol.FromV0Offset : index+protocol.FromV0Offset+protocol.FromV0Size]),
				To:            binary.BigEndian.Uint32(buf[index+protocol.ToV0Offset : index+protocol.ToV0Offset+protocol.ToV0Size]),
				Id:            binary.BigEndian.Uint32(buf[index+protocol.IdV0Offset : index+protocol.IdV0Offset+protocol.IdV0Size]),
				Operation:     binary.BigEndian.Uint32(buf[index+protocol.OperationV0Offset : index+protocol.OperationV0Offset+protocol.OperationV0Size]),
				ContentLength: binary.BigEndian.Uint64(buf[index+protocol.ContentLengthV0Offset : index+protocol.ContentLengthV0Offset+protocol.ContentLengthV0Size]),
			}
			index += protocol.MessageV0Size
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
					c.Logger().Debug().Msgf(errors.WithContext(err, PUSH).Error())
				}
			} else {
				err = c.messages.Push(&protocol.PacketV0{
					Message: &decodedMessage,
					Content: nil,
				})
				if err != nil {
					c.Logger().Debug().Msgf(errors.WithContext(err, PUSH).Error())
				}
			}
			if n == index {
				index = 0
				n, err = c.readAtLeast(buf[:cap(buf)], protocol.MessageV0Size)
				if err != nil {
					if !os.IsTimeout(err) {
						_ = c.close(err)
					}
					return
				}
			} else if n-index < protocol.MessageV0Size {
				copy(buf, buf[index:n])
				n -= index
				index = n
				n, err = c.readAtLeast(buf[n:cap(buf)], protocol.MessageV0Size-index)
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
		return nil, nil, ConnectionClosed
	}

	readPacket, err := c.messages.Pop()
	if err != nil {
		return nil, nil, errors.WithContext(err, POP)
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
		c.logger.Error().Msgf(errors.WithContext(connError, CLOSE).Error())
		c.Error = connError
		closeError := conn.Close()
		if closeError != nil && closeError != io.EOF {
			c.logger.Error().Msgf(errors.WithContext(closeError, CLOSE).Error())
		}
		return c.Error
	}
	c.Error = conn.Close()
	return c.Error
}

func (c *Conn) Close() error {
	return c.close(nil)
}

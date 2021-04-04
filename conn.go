package frisbee

import (
	"bufio"
	"encoding/binary"
	"github.com/gobwas/pool/pbufio"
	"github.com/loophole-labs/frisbee/internal/protocol"
	packetQueue "github.com/loophole-labs/frisbee/internal/ringbuffer/packet"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"io"
	"net"
	"os"
	"sync"
	"time"
)


var (
	readPool = pbufio.NewReaderPool(1024, 1<<32)
	writePool = pbufio.NewWriterPool(1024, 1<<32)
	silentLogger = zerolog.New(os.Stdout)
)

type Conn struct {
	sync.Mutex
	conn     net.Conn
	writer   *bufio.Writer
	flush    *sync.Cond
	reader   *bufio.Reader
	context  interface{}
	messages *packetQueue.RingBuffer
	closed   bool
	logger   *zerolog.Logger
	Error    error
	wg sync.WaitGroup
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
		writer:   writePool.Get(c, 1<<18),
		reader:   readPool.Get(c, 1<<22),
		messages: packetQueue.NewRingBuffer(1 << 18),
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
	//go conn.readLoop()
	go conn.fastReadLoop()

	return
}

func (c *Conn) Raw() net.Conn {
	c.wg.Done()
	c.Lock()
	defer c.Unlock()
	c.messages.Close()
	c.closed = true
	c.flush.Broadcast()
	writePool.Put(c.writer)
	readPool.Put(c.reader)
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

	var encodedMessage [protocol.HeaderLengthV0]byte
	encodedMessage[1] = protocol.Version0
	binary.BigEndian.PutUint32(encodedMessage[2:6], message.Id)
	binary.BigEndian.PutUint16(encodedMessage[6:8], message.Operation)
	binary.BigEndian.PutUint32(encodedMessage[8:12], message.Routing)
	binary.BigEndian.PutUint32(encodedMessage[12:16], message.ContentLength)
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

func (c *Conn) fastReadLoop() {
	defer c.wg.Done()
	buf := make([]byte, 1<<18, 1<<18)
	var index int
	for {
		n, err := io.ReadAtLeast(c.conn, buf[:cap(buf)], protocol.HeaderLengthV0)
		if err != nil {
			_ = c.close(err)
			return
		}
		index = 0
		for index < n {
			decodedMessage, err := protocol.DecodeV0(buf[index:protocol.HeaderLengthV0 + index])
			if err != nil {
				c.Logger().Error().Msgf("invalid buf contents, discarding")
				break
			}
			index += protocol.HeaderLengthV0
			if decodedMessage.ContentLength > 0 {
				readContent := make([]byte, decodedMessage.ContentLength, decodedMessage.ContentLength)
				if n - index < int(decodedMessage.ContentLength) {
					cp := copy(readContent, buf[index:n])
					n, err = io.ReadAtLeast(c.conn, buf[:cap(buf)], int(decodedMessage.ContentLength) - cp)
					if err != nil {
						_ = c.close(err)
						return
					}
					copy(readContent[cp:], buf[:int(decodedMessage.ContentLength) - cp])
					index = int(decodedMessage.ContentLength) - cp

					//_, err = io.ReadFull(c.conn, readContent[n - index:int(decodedMessage.ContentLength)])
					//if err != nil {
					//	_ = c.close(err)
					//	return
					//}
					//index = 0
					//n, err = io.ReadAtLeast(c.conn, buf[:cap(buf)], protocol.HeaderLengthV0)
					//if err != nil {
					//	_ = c.close(err)
					//	return
					//}
				} else {
					copy(readContent, buf[index:index + int(decodedMessage.ContentLength)])
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
				n, err = io.ReadAtLeast(c.conn, buf[:cap(buf)], protocol.HeaderLengthV0)
				if err != nil {
					_ = c.close(err)
					return
				}
			} else if n - index < protocol.HeaderLengthV0 {
				copy(buf, buf[index:n])
				n -= index
				index = n
				n, err = io.ReadAtLeast(c.conn, buf[n:cap(buf)], protocol.HeaderLengthV0 - index)
				if err != nil {
					_ = c.close(err)
					return
				}
				n += index
				index = 0
			}
		}
	}
}

//func (c *Conn) readLoop() {
//	defer c.wg.Done()
//	for {
//		message, err := c.reader.Peek(protocol.HeaderLengthV0)
//		if err != nil {
//			_ = c.close(err)
//			return
//		}
//		if message[1] != protocol.Version0 {
//			c.logger.Error().Msgf("Invalid read buffer contents, discarding")
//			_, err = c.reader.Discard(c.reader.Buffered())
//			if err != nil {
//				c.logger.Debug().Msgf("Error while discarding from read buffer: %+v", err)
//			}
//			continue
//		}
//
//		decodedMessage := protocol.MessageV0{
//			Id:            binary.BigEndian.Uint32(message[2:6]),
//			Operation:     binary.BigEndian.Uint16(message[6:8]),
//			Routing:       binary.BigEndian.Uint32(message[8:12]),
//			ContentLength: binary.BigEndian.Uint32(message[12:16]),
//		}
//
//		if decodedMessage.ContentLength > 0 {
//			content, err := c.reader.Peek(int(decodedMessage.ContentLength + protocol.HeaderLengthV0))
//			if err != nil {
//				_ = c.close(err)
//				return
//			}
//			readContent := make([]byte, decodedMessage.ContentLength)
//			copy(readContent, content[protocol.HeaderLengthV0:])
//			_, err = c.reader.Discard(int(decodedMessage.ContentLength + protocol.HeaderLengthV0))
//			if err != nil {
//				c.Logger().Debug().Msgf("Error while discarding from read buffer: %+v", err)
//			}
//			readPacket := &protocol.PacketV0{
//				Message: &decodedMessage,
//				Content: &readContent,
//			}
//			err = c.messages.Push(readPacket)
//			if err != nil {
//				c.Logger().Debug().Msgf("Error pushing read packet to message queue %+v", err)
//			}
//		} else {
//			_, err = c.reader.Discard(protocol.HeaderLengthV0)
//			if err != nil {
//				c.logger.Debug().Msgf("Error while discarding from read buffer: %+v", err)
//			}
//			readPacket := &protocol.PacketV0{
//				Message: &decodedMessage,
//				Content: nil,
//			}
//			err = c.messages.Push(readPacket)
//			if err != nil {
//				c.logger.Debug().Msgf("Error pushing read packet to message queue %+v", err)
//			}
//		}
//	}
//}

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

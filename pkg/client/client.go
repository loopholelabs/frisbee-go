package client

import (
	"bufio"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/codec"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/panjf2000/gnet/ringbuffer"
	"github.com/pkg/errors"
	"net"
	"sync"
	"time"
)

type Client struct {
	addr            string
	conn            *bufio.Reader
	bufConnWrite    *bufio.Writer
	bufConnLock     sync.Mutex
	ringBufConnRead *ringbuffer.RingBuffer
	ringBufConnLock sync.Mutex
	packets         map[uint32]*codec.Packet
	messages        chan uint32
	router          frisbee.Router
	options         *frisbee.Options
	writer          chan *[]byte
	quit            chan struct{}
}

func NewClient(addr string, router frisbee.Router, opts ...frisbee.Option) *Client {
	return &Client{
		addr:            addr,
		router:          router,
		options:         frisbee.LoadOptions(opts...),
		ringBufConnRead: ringbuffer.New(1 << 18),
		packets:         make(map[uint32]*codec.Packet),
		messages:        make(chan uint32, 4096),
		writer:          make(chan *[]byte, 4096),
		quit:            make(chan struct{}),
	}
}

func (c *Client) Connect() (err error) {
	conn, err := net.Dial("tcp4", c.addr)
	c.conn = bufio.NewReaderSize(conn, 2<<15)
	c.bufConnWrite = bufio.NewWriterSize(conn, 2<<15)
	if err == nil {
		c.options.Logger.Info().Msg("Successfully connected client")
		// Flushes the bufConnWriter
		go BufConnFlusher(&c.quit, &c.bufConnLock, c.bufConnWrite)

		// Pushes data into the ringBuffer
		go RingBufferWriter(&c.quit, &c.ringBufConnLock, c.ringBufConnRead, c.conn)

		// Reads data from the ringBuffer
		go RingBufferReader(&c.quit, &c.ringBufConnLock, c.ringBufConnRead, &c.packets, &c.messages)
	}
	return
}

func (c *Client) Stop() {
	c.bufConnLock.Lock()
	_ = c.bufConnWrite.Flush()
	close(c.quit)
}

func (c *Client) Write(message frisbee.Message, content *[]byte) error {
	if int(message.ContentLength) != len(*content) {
		return errors.New("invalid content length")
	}

	encodedMessage, err := protocol.EncodeV0(message.Id, message.Operation, message.Routing, message.ContentLength)
	if err != nil {
		return err
	}

	c.bufConnLock.Lock()
	_, _ = c.bufConnWrite.Write(encodedMessage[:])
	_, _ = c.bufConnWrite.Write(*content)
	c.bufConnLock.Unlock()
	return nil
}

func BufConnFlusher(quit *chan struct{}, bufConnLock *sync.Mutex, bufConnWriter *bufio.Writer) {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-*quit:
			return
		case <-ticker.C:
			bufConnLock.Lock()
			_ = bufConnWriter.Flush()
			bufConnLock.Unlock()
		default:
		}
	}
}

func RingBufferWriter(quit *chan struct{}, ringBufConnLock *sync.Mutex, ringBuf *ringbuffer.RingBuffer, conn *bufio.Reader) {
	for {
		select {
		case <-*quit:
			return
		default:
		}
		var data []byte
		n, err := (*conn).Read(data)
		if err == nil && n > 0 {
			ringBufConnLock.Lock()
			_, _ = ringBuf.Write(data)
			ringBufConnLock.Unlock()
		}
	}
}

func RingBufferReader(quit *chan struct{}, ringBufConnLock *sync.Mutex, ringBuf *ringbuffer.RingBuffer, packets *map[uint32]*codec.Packet, messages *chan uint32) {
	for {
		select {
		case <-*quit:
			return
		default:
		}
		if ringBuf.Length() > protocol.HeaderLengthV0 {
			if message, _ := ringBuf.LazyRead(protocol.HeaderLengthV0); len(message) == protocol.HeaderLengthV0 {
				decodedMessage, err := protocol.DecodeV0(message)
				if err != nil {
					ringBufConnLock.Lock()
					ringBuf.Reset()
					ringBufConnLock.Unlock()
					continue
				}
				packet := &codec.Packet{
					Message: &decodedMessage,
				}
				if decodedMessage.ContentLength > 0 {
					if content, _ := ringBuf.LazyRead(int(decodedMessage.ContentLength + protocol.HeaderLengthV0)); len(content) == int(decodedMessage.ContentLength+protocol.HeaderLengthV0) {
						ringBufConnLock.Lock()
						ringBuf.Shift(int(decodedMessage.ContentLength + protocol.HeaderLengthV0))
						ringBufConnLock.Unlock()

						packet.Content = content[protocol.HeaderLengthV0:]
						(*packets)[decodedMessage.Id] = packet
						*messages <- decodedMessage.Id
					}
					continue
				}
				ringBufConnLock.Lock()
				ringBuf.Shift(protocol.HeaderLengthV0)
				ringBufConnLock.Unlock()
				(*packets)[decodedMessage.Id] = packet
				*messages <- decodedMessage.Id
				continue
			}
		}
	}
}

package client

import (
	"bufio"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/codec"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/panjf2000/gnet/ringbuffer"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"net"
	"sync"
)

type Client struct {
	addr            string
	Conn            *net.TCPConn
	bufConnReader   *bufio.Reader
	bufConnWrite    *bufio.Writer
	ringBufConnRead *ringbuffer.RingBuffer
	ringBufConnLock sync.Mutex
	reactorMu       sync.Mutex
	packets         sync.Map
	messages        chan uint32
	router          frisbee.ClientRouter
	options         *Options
	writer          chan []byte
	quit            chan struct{}
}

func NewClient(addr string, router frisbee.ClientRouter, opts ...Option) *Client {
	return &Client{
		addr:            addr,
		router:          router,
		options:         LoadOptions(opts...),
		ringBufConnRead: ringbuffer.New(1 << 19),
		messages:        make(chan uint32, 1<<18),
		writer:          make(chan []byte, 1<<18),
		quit:            make(chan struct{}),
	}
}

func (c *Client) Connect() (err error) {
	c.options.Logger.Info().Msgf("Connecting to client")
	conn, err := net.Dial("tcp4", c.addr)
	c.Conn = conn.(*net.TCPConn)
	_ = c.Conn.SetNoDelay(true)
	_ = c.Conn.SetKeepAlive(true)
	_ = c.Conn.SetKeepAlivePeriod(c.options.KeepAlive)
	c.bufConnReader = bufio.NewReaderSize(conn, 1<<20)
	c.bufConnWrite = bufio.NewWriterSize(conn, 1<<20)
	if err == nil {
		c.options.Logger.Info().Msg("Successfully connected client")
		// Writes to the bufConnWrite
		go BufConnWriter(&c.quit, c.bufConnWrite, &c.writer)

		// Pushes data into the ringBuffer
		go RingBufferWriter(&c.quit, &c.ringBufConnLock, c.ringBufConnRead, c.bufConnReader)

		// Reads data from the ringBuffer
		go RingBufferReader(&c.quit, &c.ringBufConnLock, c.ringBufConnRead, &c.packets, &c.messages, c.bufConnReader)

		// Reacts to incoming messages
		go Reactor(c)
		go Reactor(c)
		go Reactor(c)

	} else {
		panic(err)
	}
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
			handlerFunc := c.router[packet.Message.Operation]
			if handlerFunc != nil {
				c.reactorMu.Lock()
				message, output, action := handlerFunc(frisbee.Message(*packet.Message), packet.Content)
				c.reactorMu.Unlock()

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

func BufConnWriter(quit *chan struct{}, bufConnWriter *bufio.Writer, writer *chan []byte) {
	for {
		select {
		case data := <-*writer:
			dataLen := len(data)
			written := 0
		FirstWrite:
			n, _ := bufConnWriter.Write((data)[written:])
			written += n
			if written != dataLen {
				goto FirstWrite
			}
			for otherData := range *writer {
				dataLen = len(otherData)
				written = 0
			LoopedWrite:
				n, _ := bufConnWriter.Write((otherData)[written:])
				written += n
				if written != dataLen {
					goto LoopedWrite
				}
				if len(*writer) < 1 {
					break
				}
			}
		case <-*quit:
			_ = bufConnWriter.Flush()
			return
		default:
			_ = bufConnWriter.Flush()
		}
	}
}

func RingBufferWriter(quit *chan struct{}, ringBufConnLock *sync.Mutex, ringBuf *ringbuffer.RingBuffer, bufConnReader *bufio.Reader) {
	var data [1 << 18]byte
	for {
		select {
		case <-*quit:
			return
		default:
			n, err := (*bufConnReader).Read(data[:])
			if err == nil && n > 0 {

				ringBufConnLock.Lock()
				_, err := ringBuf.Write(data[:n])
				ringBufConnLock.Unlock()
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

func RingBufferReader(quit *chan struct{}, ringBufConnLock *sync.Mutex, ringBuf *ringbuffer.RingBuffer, packets *sync.Map, messages *chan uint32, bufConnReader *bufio.Reader) {
	for {
		select {
		case <-*quit:
			return
		default:
			ringBufConnLock.Lock()
			if ringBuf.Length() >= protocol.HeaderLengthV0 {
				if message, _ := ringBuf.LazyRead(protocol.HeaderLengthV0); len(message) == protocol.HeaderLengthV0 {
					decodedMessage, err := protocol.DecodeV0(message)
					if err != nil {
						ringBuf.Reset()
						ringBufConnLock.Unlock()
						continue
					}
					packet := &codec.Packet{
						Message: &decodedMessage,
					}
					log.Printf("Received Message ID: %d, Len: %d, Routing: %d", decodedMessage.Id, decodedMessage.ContentLength, decodedMessage.Routing)
					if decodedMessage.ContentLength > 0 {
						if content, _ := ringBuf.LazyRead(int(decodedMessage.ContentLength + protocol.HeaderLengthV0)); len(content) == int(decodedMessage.ContentLength+protocol.HeaderLengthV0) {
							ringBuf.Shift(int(decodedMessage.ContentLength + protocol.HeaderLengthV0))
							ringBufConnLock.Unlock()
							packet.Content = content[protocol.HeaderLengthV0:]
							(*packets).Store(decodedMessage.Id, packet)
							*messages <- decodedMessage.Id
							continue
						}
						log.Printf("ELSE: %d", bufConnReader.Buffered())
						ringBufConnLock.Unlock()
						continue
					}
					ringBuf.Shift(protocol.HeaderLengthV0)
					ringBufConnLock.Unlock()
					(*packets).Store(decodedMessage.Id, packet)
					*messages <- decodedMessage.Id
					continue
				}
			}
			ringBufConnLock.Unlock()
		}
	}
}

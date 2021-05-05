package frisbee

import (
	"crypto/rand"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
	"testing"
)

func TestClientRaw(t *testing.T) {
	const testSize = 100
	const messageSize = 512
	addr := ":8192"
	clientRouter := make(ClientRouter)
	serverRouter := make(ServerRouter)

	serverIsRaw := make(chan struct{}, 1)

	serverRouter[protocol.MessagePing] = func(_ *Conn, _ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		return
	}

	var rawServerConn, rawClientConn net.Conn
	serverRouter[protocol.MessagePacket] = func(c *Conn, _ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		rawServerConn = c.Raw()
		serverIsRaw <- struct{}{}
		return
	}

	clientRouter[protocol.MessagePing] = func(_ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s := NewServer(addr, serverRouter, WithLogger(&emptyLogger))
	err := s.Start()
	assert.NoError(t, err)

	c := NewClient(addr, clientRouter, WithLogger(&emptyLogger))
	err = c.Connect()
	assert.NoError(t, err)

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	for q := 0; q < testSize; q++ {
		err := c.Write(&Message{
			Id:            uint32(q),
			Operation:     protocol.MessagePing,
			Routing:       0,
			ContentLength: messageSize,
		}, &data)
		assert.NoError(t, err)
	}

	err = c.Write(&Message{
		Id:            0,
		Operation:     protocol.MessagePacket,
		Routing:       0,
		ContentLength: 0,
	}, nil)
	assert.NoError(t, err)

	rawClientConn = c.Raw()

	<-serverIsRaw

	clientBytes := []byte("CLIENT WRITE")

	write, err := rawClientConn.Write(clientBytes)
	assert.NoError(t, err)
	assert.Equal(t, len(clientBytes), write)

	serverBuffer := make([]byte, len(clientBytes))
	read, err := rawServerConn.Read(serverBuffer)
	assert.NoError(t, err)
	assert.Equal(t, len(clientBytes), read)

	assert.Equal(t, clientBytes, serverBuffer)

	err = c.Close()
	assert.NoError(t, err)
	err = rawClientConn.Close()
	assert.NoError(t, err)

	err = s.Shutdown()
	assert.NoError(t, err)
	err = rawServerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkClientThroughput(b *testing.B) {
	const testSize = 100000
	const messageSize = 512
	addr := ":8192"
	clientRouter := make(ClientRouter)
	serverRouter := make(ServerRouter)

	serverRouter[protocol.MessagePing] = func(_ *Conn, _ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		return
	}

	clientRouter[protocol.MessagePing] = func(_ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s := NewServer(addr, serverRouter, WithLogger(&emptyLogger))
	err := s.Start()
	if err != nil {
		panic(err)
	}

	c := NewClient(addr, clientRouter, WithLogger(&emptyLogger))
	err = c.Connect()
	if err != nil {
		panic(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				err := c.Write(&Message{
					Id:            uint32(q),
					Operation:     protocol.MessagePing,
					Routing:       uint32(i),
					ContentLength: messageSize,
				}, &data)
				if err != nil {
					panic(err)
				}
			}
		}
	})
	b.StopTimer()
	err = c.Close()
	if err != nil {
		panic(err)
	}
	err = s.Shutdown()
	if err != nil {
		panic(err)
	}
}

func BenchmarkClientThroughputResponse(b *testing.B) {
	const testSize = 100000
	const messageSize = 512
	addr := ":8192"
	clientRouter := make(ClientRouter)
	serverRouter := make(ServerRouter)

	finished := make(chan struct{}, 1)

	serverRouter[protocol.MessagePing] = func(_ *Conn, incomingMessage Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		if incomingMessage.Id == testSize-1 {
			outgoingMessage = &Message{
				Id:            testSize,
				Operation:     protocol.MessagePong,
				Routing:       0,
				ContentLength: 0,
			}
		}
		return
	}

	clientRouter[protocol.MessagePong] = func(incomingMessage Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		if incomingMessage.Id == testSize {
			finished <- struct{}{}
		}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s := NewServer(addr, serverRouter, WithLogger(&emptyLogger))
	err := s.Start()
	if err != nil {
		panic(err)
	}

	c := NewClient(addr, clientRouter, WithLogger(&emptyLogger))
	err = c.Connect()
	if err != nil {
		panic(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				err := c.Write(&Message{
					Id:            uint32(q),
					Operation:     protocol.MessagePing,
					Routing:       uint32(i),
					ContentLength: messageSize,
				}, &data)
				if err != nil {
					panic(err)
				}
			}
			<-finished
		}
	})
	b.StopTimer()
	err = c.Close()
	if err != nil {
		panic(err)
	}
	err = s.Shutdown()
	if err != nil {
		panic(err)
	}
}

package frisbee

import (
	"crypto/rand"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

func TestServerRaw(t *testing.T) {
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

	serverBytes := []byte("SERVER WRITE")

	write, err := rawServerConn.Write(serverBytes)
	assert.NoError(t, err)
	assert.Equal(t, len(serverBytes), write)

	clientBuffer := make([]byte, len(serverBytes))
	read, err := rawClientConn.Read(clientBuffer)
	assert.NoError(t, err)
	assert.Equal(t, len(serverBytes), read)

	assert.Equal(t, serverBytes, clientBuffer)

	err = c.Close()
	assert.NoError(t, err)
	err = rawClientConn.Close()
	assert.NoError(t, err)

	err = s.Shutdown()
	assert.NoError(t, err)
	err = rawServerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkThroughput(b *testing.B) {
	const testSize = 100000
	const messageSize = 512
	addr := ":8192"
	router := make(ServerRouter)

	router[protocol.MessagePing] = func(_ *Conn, _ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	server := NewServer(addr, router, WithLogger(&emptyLogger))
	err := server.Start()
	if err != nil {
		log.Printf("Could not start server")
		panic(err)
	}

	frisbeeConn, err := Connect("tcp", addr, time.Minute*3, nil)
	if err != nil {
		log.Printf("Could not connect to server")
		panic(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				err := frisbeeConn.Write(&Message{
					Id:            uint32(q),
					Operation:     protocol.MessagePing,
					Routing:       uint32(i),
					ContentLength: messageSize,
				}, &data)
				if err != nil {
					log.Printf("Could not write to server")
					panic(err)
				}
			}
		}
	})
	b.StopTimer()
	err = frisbeeConn.Close()
	if err != nil {
		log.Printf("Could not disconnect from server")
		panic(err)
	}
	err = server.Shutdown()
	if err != nil {
		log.Printf("Could not shut down server")
		panic(err)
	}
}

func BenchmarkThroughputWithResponse(b *testing.B) {
	const testSize = 100000
	const messageSize = 512
	addr := ":8192"
	router := make(ServerRouter)

	router[protocol.MessagePing] = func(_ *Conn, incomingMessage Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
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

	emptyLogger := zerolog.New(ioutil.Discard)
	server := NewServer(addr, router, WithLogger(&emptyLogger))
	err := server.Start()
	if err != nil {
		log.Printf("Could not start server")
		panic(err)
	}

	frisbeeConn, err := Connect("tcp", addr, time.Minute*3, nil)
	if err != nil {
		log.Printf("Could not connect to server")
		panic(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				err := frisbeeConn.Write(&Message{
					Id:            uint32(q),
					Operation:     protocol.MessagePing,
					Routing:       uint32(i),
					ContentLength: messageSize,
				}, &data)
				if err != nil {
					log.Printf("Could not write to server")
					panic(err)
				}
			}
			message, _, err := frisbeeConn.Read()
			if err != nil {
				panic(err)
			}

			if message.Id != testSize {
				log.Printf("Could retrieve data from server")
				panic(errors.New("invalid decoded message id"))
			}
		}

	})
	b.StopTimer()
	err = frisbeeConn.Close()
	if err != nil {
		log.Printf("Could not disconnect from server")
		panic(err)
	}
	err = server.Shutdown()
	if err != nil {
		log.Printf("Could not shut down server")
		panic(err)
	}
}

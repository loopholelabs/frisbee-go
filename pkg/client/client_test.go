package client

import (
	"crypto/rand"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/loophole-labs/frisbee/pkg/server"
	"github.com/rs/zerolog"
	"io/ioutil"
	"testing"
)

func BenchmarkClientThroughput(b *testing.B) {
	const testSize = 100000
	const messageSize = 2048
	addr := ":8192"
	serverRouter := make(frisbee.ServerRouter)
	clientRouter := make(frisbee.ClientRouter)

	serverRouter[protocol.MessagePing] = func(_ frisbee.Conn, _ frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	clientRouter[protocol.MessagePing] = func(_ frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s := server.NewServer(addr, serverRouter, server.WithAsync(true), server.WithLogger(&emptyLogger), server.WithMulticore(true), server.WithLoops(16))
	s.Start()

	c := NewClient("127.0.0.1:8192", clientRouter, WithLogger(&emptyLogger))
	_ = c.Connect()

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				err := c.Write(frisbee.Message{
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
	_ = c.Stop()
	_ = s.Stop()
}

func BenchmarkClientThroughputResponse(b *testing.B) {
	const testSize = 100000
	const messageSize = 512

	finished := make(chan struct{})

	addr := ":8192"
	serverRouter := make(frisbee.ServerRouter)
	serverRouter[protocol.MessagePing] = func(_ frisbee.Conn, incomingMessage frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		if incomingMessage.Id == testSize-1 {
			outgoingMessage = &frisbee.Message{
				Id:            testSize,
				Operation:     protocol.MessagePong,
				Routing:       0,
				ContentLength: 0,
			}
		}
		return
	}

	clientRouter := make(frisbee.ClientRouter)
	clientRouter[protocol.MessagePong] = func(incomingMessage frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		if incomingMessage.Id == testSize {
			finished <- struct{}{}
		}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s := server.NewServer(addr, serverRouter, server.WithAsync(true), server.WithLogger(&emptyLogger), server.WithMulticore(true), server.WithLoops(16))
	s.Start()

	c := NewClient("127.0.0.1:8192", clientRouter, WithLogger(&emptyLogger))
	_ = c.Connect()

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				err := c.Write(frisbee.Message{
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
	_ = c.Stop()
	_ = s.Stop()
}

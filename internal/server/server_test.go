package server

import (
	"crypto/rand"
	"github.com/loophole-labs/frisbee/internal/conn"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"testing"
)

func BenchmarkThroughput(b *testing.B) {
	const testSize = 10000
	const messageSize = 512
	addr := ":8192"
	router := make(Router)

	router[protocol.MessagePing] = func(_ *conn.Conn, _ protocol.MessageV0, _ []byte) (outgoingMessage *protocol.MessageV0, outgoingContent []byte, action int) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	server := NewServer(addr, router, WithLogger(&emptyLogger))
	err := server.Start()
	if err != nil {
		log.Printf("Could not start server")
		panic(err)
	}

	frisbeeConn, err := conn.Connect("tcp", addr)
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
				err := frisbeeConn.Write(&protocol.MessageV0{
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
	const testSize = 10000
	const messageSize = 512
	addr := ":8192"
	router := make(Router)

	router[protocol.MessagePing] = func(_ *conn.Conn, incomingMessage protocol.MessageV0, _ []byte) (outgoingMessage *protocol.MessageV0, outgoingContent []byte, action int) {
		if incomingMessage.Id == testSize-1 {
			outgoingMessage = &protocol.MessageV0{
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

	frisbeeConn, err := conn.Connect("tcp", addr)
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
				err := frisbeeConn.Write(&protocol.MessageV0{
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

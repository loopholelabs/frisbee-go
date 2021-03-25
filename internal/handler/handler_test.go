package handler

import (
	"bufio"
	"crypto/rand"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"io"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

func BenchmarkThroughput(b *testing.B) {
	const testSize = 10000
	const messageSize = 512
	const bufferSize = messageSize << 9
	addr := "0.0.0.0:8192"
	router := make(frisbee.ServerRouter)

	router[protocol.MessagePing] = func(_ frisbee.Conn, _ frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	started := make(chan struct{})
	emptyLogger := zerolog.New(ioutil.Discard)
	handler := StartHandler(started, addr, true, true, 16, time.Minute*5, &emptyLogger, router, nil, nil, nil, nil, nil, nil)
	<-started

	tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8192")
	if err != nil {
		panic(err)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		panic(err)
	}

	bufConn := bufio.NewWriterSize(conn, bufferSize)
	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				encodedMessage, _ := protocol.EncodeV0(uint32(q), protocol.MessagePing, uint32(i), messageSize)
				_, err = bufConn.Write(encodedMessage[:])
				if err != nil {
					panic(err)
				}
				_, err = bufConn.Write(data)
				if err != nil {
					panic(err)
				}
			}
			err = bufConn.Flush()
			if err != nil {
				panic(err)
			}
		}

	})
	b.StopTimer()
	err = conn.Close()
	if err != nil {
		panic(err)
	}
	_ = handler.Stop()
}

func BenchmarkThroughputWithResponse(b *testing.B) {
	const testSize = 10000
	const messageSize = 512
	const bufferSize = messageSize << 9
	addr := "0.0.0.0:8192"
	router := make(frisbee.ServerRouter)

	router[protocol.MessagePing] = func(_ frisbee.Conn, incomingMessage frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
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

	started := make(chan struct{})
	emptyLogger := zerolog.New(ioutil.Discard)
	handler := StartHandler(started, addr, true, true, 16, time.Minute*5, &emptyLogger, router, nil, nil, nil, nil, nil, nil)
	<-started

	tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8192")
	if err != nil {
		panic(err)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		panic(err)
	}

	response := [protocol.HeaderLengthV0]byte{}
	bufConn := bufio.NewWriterSize(conn, bufferSize)
	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				encodedMessage, _ := protocol.EncodeV0(uint32(q), protocol.MessagePing, uint32(i), messageSize)
				_, err = bufConn.Write(encodedMessage[:])
				if err != nil {
					panic(err)
				}
				_, err = bufConn.Write(data)
				if err != nil {
					panic(err)
				}
			}
			err = bufConn.Flush()
			if err != nil {
				panic(err)
			}
			_, err = io.ReadFull(conn, response[:])
			if err != nil {
				panic(err)
			}
			decodedMessage, err := protocol.DecodeV0(response[:])
			if err != nil {
				panic(err)
			}
			if decodedMessage.Id != testSize {
				panic(errors.New("invalid decoded message id"))
			}
		}

	})
	b.StopTimer()
	err = conn.Close()
	if err != nil {
		panic(err)
	}
	_ = handler.Stop()
}

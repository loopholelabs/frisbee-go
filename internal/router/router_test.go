package router

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/panjf2000/gnet"
	"github.com/pkg/errors"
	"io"
	"net"
	"testing"
)

func BenchmarkThroughput(b *testing.B) {
	const testSize = 10000
	const messageSize = 32
	const bufferSize = messageSize << 9
	addr := fmt.Sprintf("0.0.0.0:8192")
	messageMap := make(MessageMap)

	messageMap[protocol.MessagePing] = func(message protocol.MessageV0, content []byte) ([]byte, int) {
		return nil, 0
	}

	started := make(chan struct{})

	go StartServer(started, addr, true, true, messageMap)
	<-started

	tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8192")
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		panic(err)
	}

	bufConn := bufio.NewWriterSize(conn, bufferSize)
	data := make([]byte, 32)
	_, _ = rand.Read(data)

	b.Run("client-test", func(b *testing.B) {
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
	_ = gnet.Stop(context.Background(), addr)
}

func BenchmarkThroughputWithResponse(b *testing.B) {
	const testSize = 10000
	const messageSize = 32
	const bufferSize = messageSize << 9
	addr := fmt.Sprintf("0.0.0.0:8192")
	messageMap := make(MessageMap)

	messageMap[protocol.MessagePing] = func(message protocol.MessageV0, content []byte) ([]byte, int) {
		if message.Id == testSize-1 {
			encodedMessage, _ := protocol.EncodeV0(testSize, protocol.MessagePong, 0, 0)
			return encodedMessage[:], 0
		}
		return nil, 0
	}

	started := make(chan struct{})

	go StartServer(started, addr, true, true, messageMap)
	<-started

	tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8192")
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		panic(err)
	}

	response := [protocol.HeaderLengthV0]byte{}
	bufConn := bufio.NewWriterSize(conn, bufferSize)
	data := make([]byte, 32)
	_, _ = rand.Read(data)

	b.Run("client-test", func(b *testing.B) {
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
	_ = gnet.Stop(context.Background(), addr)
}

package router

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"net"
	"testing"
)

func BenchmarkThroughput(b *testing.B) {
	addr := fmt.Sprintf("0.0.0.0:8192")
	messageMap := make(MessageMap)

	messageMap[protocol.MessagePing] = func(message protocol.MessageV0, content []byte) ([]byte, int) {
		return nil, 0
	}

	go StartServer(addr, true, true, messageMap)

	tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8192")
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	bufConn := bufio.NewWriterSize(conn, 2<<12)
	data := make([]byte, 32)
	_, _ = rand.Read(data)

	b.Run("client-test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for i := 0; i < 10000; i++ {
				encodedMessage, _ := protocol.EncodeV0(uint16(i), protocol.MessagePing, 0, 512)
				_, _ = bufConn.Write(encodedMessage[:])
				_, _ = bufConn.Write(data)
			}
			_ = bufConn.Flush()
		}
	})
}

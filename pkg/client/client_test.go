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
	const testSize = 10000
	const messageSize = 512
	addr := "0.0.0.0:8192"
	router := make(frisbee.Router)

	router[protocol.MessagePing] = func(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}
	emptyLogger := zerolog.New(ioutil.Discard)
	s := server.NewServer(addr, router, frisbee.WithAsync(true), frisbee.WithLogger(&emptyLogger), frisbee.WithMulticore(true), frisbee.WithLoops(16))
	go s.Start()

	c := NewClient("127.0.0.1:8192", router, frisbee.WithLogger(&emptyLogger))
	_ = c.Connect()

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("client-test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				_ = c.Write(frisbee.Message{
					Id:            uint32(q),
					Operation:     protocol.MessagePing,
					Routing:       uint32(i),
					ContentLength: messageSize,
				}, &data)
			}
		}

	})
	b.StopTimer()
	c.Stop()
	_ = s.Stop()
}

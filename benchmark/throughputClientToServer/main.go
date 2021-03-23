package main

import (
	"crypto/rand"
	"fmt"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/loophole-labs/frisbee/pkg/client"
	"github.com/loophole-labs/frisbee/pkg/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"time"
)

const testSize = 10000
const messageSize = 512
const runs = 100
const port = 8192

var complete = make(chan struct{})

func main() {
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
			complete <- struct{}{}
		}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s := server.NewServer(fmt.Sprintf(":%d", port), serverRouter, server.WithAsync(true), server.WithLogger(&emptyLogger), server.WithMulticore(true), server.WithLoops(16))
	s.Start()

	c := client.NewClient(fmt.Sprintf("127.0.0.1:%d", port), clientRouter, client.WithLogger(&emptyLogger))
	_ = c.Connect()

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	duration := time.Nanosecond
	for i := 1; i < runs+1; i++ {
		start := time.Now()
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
		<-complete
		runTime := time.Since(start)
		log.Printf("Benchmark Time for test %d: %d ns", i, runTime.Nanoseconds())
		duration += runTime
	}
	log.Printf("Average Benchmark time for %d runs: %d ns", runs, duration.Nanoseconds()/runs)
	_ = c.Stop()
	_ = s.Stop()
}

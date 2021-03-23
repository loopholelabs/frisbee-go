package main

import (
	"crypto/rand"
	"fmt"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/loophole-labs/frisbee/pkg/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"time"
)

const testSize = 100000
const messageSize = 512
const runs = 10
const port = 8192

var complete = make(chan struct{})

func main() {
	router := make(frisbee.ServerRouter)
	router[protocol.MessagePong] = func(_ frisbee.Conn, incomingMessage frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		if incomingMessage.Id == testSize {
			complete <- struct{}{}
		}
		return
	}

	var benchmarkConnection frisbee.Conn
	connected := make(chan struct{})

	emptyLogger := zerolog.New(ioutil.Discard)
	s := server.NewServer(fmt.Sprintf(":%d", port), router, server.WithAsync(true), server.WithLogger(&emptyLogger), server.WithMulticore(true), server.WithLoops(16))
	s.UserOnOpened = func(s *server.Server, c frisbee.Conn) frisbee.Action {
		benchmarkConnection = c
		connected <- struct{}{}
		return frisbee.None
	}

	s.Start()

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	duration := time.Nanosecond * 0
	<-connected
	for i := 1; i < runs+1; i++ {
		start := time.Now()
		for q := 1; q < testSize+1; q++ {
			err := benchmarkConnection.Write(frisbee.Message{
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
	_ = s.Stop()
}

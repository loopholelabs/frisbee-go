package main

import (
	"crypto/rand"
	"fmt"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/protocol"
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

func handlePong(incomingMessage frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if incomingMessage.Id == testSize {
		complete <- struct{}{}
	}
	return
}

func main() {
	router := make(frisbee.ClientRouter)
	router[protocol.MessagePong] = handlePong

	emptyLogger := zerolog.New(ioutil.Discard)

	c := frisbee.NewClient(fmt.Sprintf("127.0.0.1:%d", port), router, frisbee.WithLogger(&emptyLogger))
	_ = c.Connect()

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	duration := time.Nanosecond * 0
	for i := 1; i < runs+1; i++ {
		start := time.Now()
		for q := 0; q < testSize; q++ {
			err := c.Write(&frisbee.Message{
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
	log.Printf("Average Benchmark time for %d runs: %d ns, throughput: %f mb/s", runs, duration.Nanoseconds()/runs, (1/((duration.Seconds()/runs)/testSize)*messageSize)/(1024*1024))
	_ = c.Close()
}

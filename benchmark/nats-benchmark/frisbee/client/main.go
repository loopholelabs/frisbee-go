package main

import (
	"crypto/rand"
	"fmt"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/loov/hrtime"
	"github.com/rs/zerolog"
	"io/ioutil"
	"log"
)

const testSize = 100000
const messageSize = 512
const runs = 100
const port = 8192

func main() {
	router := make(frisbee.ClientRouter)
	emptyLogger := zerolog.New(ioutil.Discard)
	c := frisbee.NewClient(fmt.Sprintf("127.0.0.1:%d", port), router, frisbee.WithLogger(&emptyLogger))
	_ = c.Connect()

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	i := 0
	bench := hrtime.NewBenchmark(runs)
	for bench.Next() {
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
		i++
	}
	log.Println(bench.Histogram(10))
	_ = c.Close()
}

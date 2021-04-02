package main

import (
	"crypto/rand"
	"github.com/loophole-labs/frisbee"
	"github.com/loov/hrtime"
	"hash/crc32"
	"os"
	"os/signal"
)

const PUB = uint16(1)
const testSize = 100000
const messageSize = 512
const runs = 100

//var complete = make(chan struct{})

var topic = []byte("SENDING")
var topicHash = crc32.ChecksumIEEE(topic)

func main() {
	router := make(frisbee.ClientRouter)
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)

	c := frisbee.NewClient("127.0.0.1:8192", router)
	err := c.Connect()
	if err != nil {
		panic(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	go func() {
		i := 0
		bench := hrtime.NewBenchmark(runs)
		for bench.Next() {
			for q := 0; q < testSize; q++ {
				err := c.Write(&frisbee.Message{
					Id:            uint32(i),
					Operation:     PUB,
					Routing:       topicHash,
					ContentLength: uint32(len(data)),
				}, &data)
				if err != nil {
					panic(err)
				}
			}
			i++

		}
	}()

	<-exit
	err = c.Close()
	if err != nil {
		panic(err)
	}
}

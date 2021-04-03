package main

import (
	"crypto/rand"
	"github.com/loov/hrtime"
	"github.com/nats-io/nats.go"
	"log"
)

const testSize = 100000
const messageSize = 2048
const runs = 100

var complete = make(chan struct{})

func main() {
	nc, _ := nats.Connect(nats.DefaultURL)

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	_, _ = nc.Subscribe("DONE", func(m *nats.Msg) {
		complete <- struct{}{}
	})

	i := 0
	bench := hrtime.NewBenchmark(runs)
	for bench.Next() {
		for q := 0; q < testSize; q++ {
			err := nc.Publish("BENCH", data)
			if err != nil {
				panic(err)
			}
		}
		err := nc.Publish("BENCH", []byte("END"))
		if err != nil {
			panic(err)
		}
		<-complete
		i++
	}
	log.Println(bench.Histogram(10))
	nc.Close()
}

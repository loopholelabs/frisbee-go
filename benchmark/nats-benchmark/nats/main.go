package main

import (
	"crypto/rand"
	"github.com/loov/hrtime"
	"github.com/nats-io/nats.go"
	"log"
)

const testSize = 100000
const messageSize = 512
const runs = 100

func main() {
	nc, _ := nats.Connect(nats.DefaultURL)

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	i := 0
	bench := hrtime.NewBenchmark(runs)
	for bench.Next() {
		for q := 0; q < testSize; q++ {
			err := nc.Publish("bench", data)
			if err != nil {
				panic(err)
			}
		}
		i++
	}
	log.Println(bench.Histogram(10))
	nc.Close()
}

package main

import (
	"github.com/nats-io/nats.go"
	"os"
	"os/signal"
)

func main() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)

	nc, _ := nats.Connect(nats.DefaultURL)

	_, _ = nc.Subscribe("BENCH", func(m *nats.Msg) {
		if string(m.Data) == "END" {
			_ = nc.Publish("DONE", []byte("END"))
		}
	})

	<-exit
	nc.Close()
}

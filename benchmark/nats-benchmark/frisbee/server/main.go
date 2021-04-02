package main

import (
	"fmt"
	"github.com/loophole-labs/frisbee"
	"github.com/rs/zerolog"
	"io/ioutil"
	"os"
	"os/signal"
)

const port = 8192

func main() {
	router := make(frisbee.ServerRouter)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)

	emptyLogger := zerolog.New(ioutil.Discard)

	s := frisbee.NewServer(fmt.Sprintf(":%d", port), router, frisbee.WithLogger(&emptyLogger))
	_ = s.Start()

	<-exit
	err := s.Shutdown()
	if err != nil {
		panic(err)
	}
}

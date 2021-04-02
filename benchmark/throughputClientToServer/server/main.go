package main

import (
	"fmt"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/rs/zerolog"
	"io/ioutil"
	"os"
	"os/signal"
)

const testSize = 100000
const port = 8192

func handlePing(_ *frisbee.Conn, incomingMessage frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
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

func main() {
	router := make(frisbee.ServerRouter)
	router[protocol.MessagePing] = handlePing
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)

	emptyLogger := zerolog.New(ioutil.Discard)

	s := frisbee.NewServer(fmt.Sprintf(":%d", port), router, frisbee.WithLogger(&emptyLogger))
	s.UserOnOpened = func(server *frisbee.Server, c *frisbee.Conn) frisbee.Action {
		server.Options.Logger.Debug().Msgf("Client connected: %s", c.RemoteAddr())
		return frisbee.None
	}

	_ = s.Start()

	<-exit
	err := s.Shutdown()
	if err != nil {
		panic(err)
	}
}

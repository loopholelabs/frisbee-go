package frisbee_test

import (
	"github.com/loophole-labs/frisbee"
	"github.com/rs/zerolog"
	"os"
)

func ExampleNewClient() {
	router := make(frisbee.ClientRouter)

	router[0] = func(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	frisbee.NewClient("127.0.0.1:8080", router, frisbee.WithLogger(&logger))
}

func ExampleNewServer() {
	router := make(frisbee.ServerRouter)

	router[0] = func(c *frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	frisbee.NewServer("127.0.0.1:8080", router, frisbee.WithLogger(&logger))
}

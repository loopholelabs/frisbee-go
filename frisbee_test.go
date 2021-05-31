package frisbee_test

import (
	"github.com/loophole-labs/frisbee"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

func ExampleClient_Connect() {
	router := make(frisbee.ClientRouter)

	router[0] = func(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	client := frisbee.NewClient("127.0.0.1:8080", router, frisbee.WithLogger(&logger))

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}

func ExampleClient_Write() {
	router := make(frisbee.ClientRouter)

	router[0] = func(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	client := frisbee.NewClient("127.0.0.1:8080", router, frisbee.WithLogger(&logger))

	err := client.Connect()
	if err != nil {
		panic(err)
	}

	data := []byte("TEST DATA")

	err = client.Write(&frisbee.Message{
		From:          uint32(16),
		To:            uint32(32),
		Id:            uint32(32),
		Operation:     uint32(0),
		ContentLength: uint64(len(data)),
	}, &data)

	if err != nil {
		panic(err)
	}
}

func ExampleClient_Raw() {
	router := make(frisbee.ClientRouter)

	router[0] = func(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	client := frisbee.NewClient("127.0.0.1:8080", router, frisbee.WithLogger(&logger))

	err := client.Connect()
	if err != nil {
		panic(err)
	}

	data := []byte("TEST DATA")

	err = client.Write(&frisbee.Message{
		From:          uint32(16),
		To:            uint32(32),
		Id:            uint32(32),
		Operation:     uint32(0),
		ContentLength: uint64(len(data)),
	}, &data)

	if err != nil {
		panic(err)
	}

	rawTCPConnection, err := client.Raw()
	if err != nil {
		panic(err)
	}

	n, err := rawTCPConnection.Write(data)
	if err != nil {
		return
	}

	log.Printf("Wrote: %d bytes", n)
	// Output: Wrote 9 bytes
}

func ExampleNewServer() {
	router := make(frisbee.ServerRouter)

	router[0] = func(c *frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	frisbee.NewServer("127.0.0.1:8080", router, frisbee.WithLogger(&logger))
}

func ExampleServer_Start() {
	router := make(frisbee.ServerRouter)

	router[0] = func(c *frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	server := frisbee.NewServer("127.0.0.1:8080", router, frisbee.WithLogger(&logger))

	err := server.Start()
	if err != nil {
		panic(err)
	}
}

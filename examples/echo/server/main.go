package main

import (
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/pkg/server"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
)

const PING = uint16(1)
const PONG = uint16(2)

func handlePing(_ frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if incomingMessage.ContentLength > 0 {
		log.Printf("Server Received Message: %s", incomingContent)
		outgoingMessage = &frisbee.Message{
			Id:            incomingMessage.Id,
			Routing:       incomingMessage.Routing,
			Operation:     PONG,
			ContentLength: incomingMessage.ContentLength,
		}
		outgoingContent = incomingContent
	}

	return
}

func main() {
	router := make(frisbee.ServerRouter)
	router[PING] = handlePing
	exit := make(chan os.Signal)
	signal.Notify(exit, os.Interrupt)

	s := server.NewServer(":8192", router)
	s.Start()

	<-exit
	err := s.Stop()
	if err != nil {
		panic(err)
	}
}

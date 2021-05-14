package main

import (
	"github.com/loophole-labs/frisbee"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
)

const PING = uint32(1)
const PONG = uint32(2)

func handlePing(_ *frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if incomingMessage.ContentLength > 0 {
		log.Printf("Server Received Message: %s", incomingContent)
		outgoingMessage = &frisbee.Message{
			From:          incomingMessage.From,
			To:            incomingMessage.To,
			Id:            incomingMessage.Id,
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

	s := frisbee.NewServer(":8192", router)
	err := s.Start()
	if err != nil {
		panic(err)
	}

	<-exit
	err = s.Shutdown()
	if err != nil {
		panic(err)
	}
}

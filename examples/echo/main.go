package main

import (
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/pkg/server"
)

const PING = uint16(1)
const PONG = uint16(2)

func handlePing(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	outgoingMessage = &frisbee.Message{
		Id:            incomingMessage.Id,
		Routing:       incomingMessage.Routing,
		Operation:     PONG,
		ContentLength: uint32(len(incomingContent)),
	}
	outgoingContent = incomingContent
	return
}

func main() {
	router := make(frisbee.Router)
	router[PING] = handlePing

	s := server.NewServer(":8192", router)
	_ = s.Start()
}

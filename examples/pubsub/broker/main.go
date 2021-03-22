package main

import (
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/pkg/server"
	"github.com/rs/zerolog/log"
	"hash/crc32"
	"os"
	"os/signal"
)

const PUB = uint16(1)
const SUB = uint16(2)

var subscribers = make(map[uint32][]frisbee.Conn)

func handleSub(c frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if incomingMessage.ContentLength > 0 {
		log.Printf("Server Received SUB on topic %s from %s", string(incomingContent), c.RemoteAddr())
		checksum := crc32.ChecksumIEEE(incomingContent)
		subscribers[checksum] = append(subscribers[checksum], c)
	}
	return
}

func handlePub(_ frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if incomingMessage.ContentLength > 0 {
		log.Printf("Server Received PUB on hashed topic %d with content %s", incomingMessage.Routing, string(incomingContent))
		if connections := subscribers[incomingMessage.Routing]; connections != nil {
			for _, c := range connections {
				_ = c.Write(frisbee.Message{
					Id:            0,
					Operation:     PUB,
					Routing:       incomingMessage.Routing,
					ContentLength: incomingMessage.ContentLength,
				}, &incomingContent)
			}
		}
	}

	return
}

func main() {
	router := make(frisbee.ServerRouter)
	router[SUB] = handleSub
	router[PUB] = handlePub
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

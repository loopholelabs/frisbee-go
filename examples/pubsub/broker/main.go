package main

import (
	"github.com/loophole-labs/frisbee"
	pubsub "github.com/loophole-labs/frisbee/examples/pubsub/schema"
	"github.com/rs/zerolog/log"
	"hash/crc32"
	"os"
	"os/signal"
)

const PUB = uint32(1)
const SUB = uint32(2)

var subscribers = make(map[uint32][]*frisbee.Conn)

type ServerHandler struct{}

func (ServerHandler) HandleSub(c *frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if incomingMessage.ContentLength > 0 {
		log.Printf("Server Received SUB on topic %s from %s", string(incomingContent), c.RemoteAddr())
		checksum := crc32.ChecksumIEEE(incomingContent)
		subscribers[checksum] = append(subscribers[checksum], c)
	}
	return
}

func (ServerHandler) HandlePub(_ *frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if incomingMessage.ContentLength > 0 {
		log.Printf("Server Received PUB on hashed topic %d with content %s", incomingMessage.From, string(incomingContent))
		if connections := subscribers[incomingMessage.From]; connections != nil {
			for _, c := range connections {
				_ = c.Write(&frisbee.Message{
					From:          incomingMessage.From,
					To:            incomingMessage.To,
					Id:            0,
					Operation:     PUB,
					ContentLength: incomingMessage.ContentLength,
				}, &incomingContent)
			}
		}
	}

	return
}

func main() {
	exit := make(chan os.Signal)
	signal.Notify(exit, os.Interrupt)
	s := pubsub.NewServer(":8192", &ServerHandler{})
	_ = s.Start()

	<-exit
	err := s.Shutdown()
	if err != nil {
		panic(err)
	}
}

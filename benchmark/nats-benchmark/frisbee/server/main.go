package main

import (
	"github.com/loophole-labs/frisbee"
	"hash/crc32"
	"os"
	"os/signal"
)

const PUB = uint32(1)
const SUB = uint32(2)

var subscribers = make(map[uint32][]*frisbee.Conn)

func handleSub(c *frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if incomingMessage.ContentLength > 0 {
		checksum := crc32.ChecksumIEEE(incomingContent)
		subscribers[checksum] = append(subscribers[checksum], c)
	}
	return
}

func handlePub(_ *frisbee.Conn, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if connections := subscribers[incomingMessage.To]; connections != nil {
		for _, c := range connections {
			_ = c.Write(&frisbee.Message{
				To:            incomingMessage.To,
				From:          incomingMessage.From,
				Id:            0,
				Operation:     PUB,
				ContentLength: incomingMessage.ContentLength,
			}, &incomingContent)
		}
	}

	return
}

func main() {
	router := make(frisbee.ServerRouter)
	router[SUB] = handleSub
	router[PUB] = handlePub
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)

	s := frisbee.NewServer(":8192", router)
	_ = s.Start()

	<-exit
	err := s.Shutdown()
	if err != nil {
		panic(err)
	}
}

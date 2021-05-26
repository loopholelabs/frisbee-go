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

var topic = []byte("TOPIC 1")
var topicHash = crc32.ChecksumIEEE(topic)

type ClientHandler struct{}

// Handle the PUB message type
func (ClientHandler) HandlePub(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if incomingMessage.From == topicHash {
		log.Printf("Client Received Message on Topic %s: %s", string(topic), string(incomingContent))
	}
	return
}

func (ClientHandler) HandleSub(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	panic("panic")
}
func main() {
	exit := make(chan os.Signal)
	signal.Notify(exit, os.Interrupt)

	c := pubsub.NewClient("127.0.0.1:8192", &ClientHandler{})
	err := c.Connect()
	if err != nil {
		panic(err)
	}

	i := 0

	// First subscribe to the topic
	err = c.Write(&frisbee.Message{
		From:          0,
		To:            0,
		Id:            uint32(i),
		Operation:     SUB,
		ContentLength: uint64(len(topic)),
	}, &topic)
	if err != nil {
		panic(err)
	}

	// Now the handle pub function will be called
	// automatically whenever a message that matches the topic arrives

	<-exit
	err = c.Close()
	if err != nil {
		panic(err)
	}
}

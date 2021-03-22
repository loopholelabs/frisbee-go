package main

import (
	"fmt"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/pkg/client"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"time"
)

const PING = uint16(1)
const PONG = uint16(2)

func handlePong(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	if incomingMessage.ContentLength > 0 {
		log.Printf("Client Received Message: %s", string(incomingContent))
	}
	return
}

func main() {
	router := make(frisbee.Router)
	router[PONG] = handlePong
	exit := make(chan os.Signal)
	signal.Notify(exit, os.Interrupt)

	c := client.NewClient("127.0.0.1:8192", router)
	err := c.Connect()
	if err != nil {
		panic(err)
	}

	go func() {
		i := 0
		for {
			message := []byte(fmt.Sprintf("ECHO MESSAGE: %d", i))
			err := c.Write(frisbee.Message{
				Id:            uint32(i),
				Operation:     PING,
				Routing:       0,
				ContentLength: uint32(len(message)),
			}, &message)
			if err != nil {
				panic(err)
			}
			i++
			time.Sleep(time.Second)
		}
	}()

	<-exit
	err = c.Stop()
	if err != nil {
		panic(err)
	}
}

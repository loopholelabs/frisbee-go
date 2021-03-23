package main

import (
	"fmt"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/loophole-labs/frisbee/pkg/client"
	"github.com/rs/zerolog"
	"io/ioutil"
	"os"
	"os/signal"
)

const testSize = 100000
const port = 8192

func main() {
	router := make(frisbee.ClientRouter)
	exit := make(chan os.Signal)
	signal.Notify(exit, os.Interrupt)

	router[protocol.MessagePing] = func(incomingMessage frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		if incomingMessage.Id == testSize-1 {
			outgoingMessage = &frisbee.Message{
				Id:            testSize,
				Operation:     protocol.MessagePong,
				Routing:       0,
				ContentLength: 0,
			}
		}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)

	c := client.NewClient(fmt.Sprintf("127.0.0.1:%d", port), router, client.WithLogger(&emptyLogger))
	_ = c.Connect()

	<-exit
	err := c.Stop()
	if err != nil {
		panic(err)
	}
}

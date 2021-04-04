package main

import (
	"crypto/rand"
	"github.com/loophole-labs/frisbee"
	"github.com/loov/hrtime"
	"hash/crc32"
	"log"
)

const PUB = uint16(1)
const SUB = uint16(2)
const testSize = 100000
const messageSize = 2048
const runs = 100

var complete = make(chan struct{})

var topic = []byte("SENDING")
var topicHash = crc32.ChecksumIEEE(topic)

var receiveTopic = []byte("RECEIVING")
var receiveTopicHash = crc32.ChecksumIEEE(receiveTopic)

func handlePub(incomingMessage frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
	log.Printf("RECEIVED MESSAGE")
	if incomingMessage.Routing == receiveTopicHash {
		complete <- struct{}{}
	}
	return
}

func main() {
	router := make(frisbee.ClientRouter)

	router[PUB] = handlePub

	c := frisbee.NewClient("127.0.0.1:8192", router)
	err := c.Connect()
	if err != nil {
		panic(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	END := []byte("END")

	err = c.Write(&frisbee.Message{
		Id:            0,
		Operation:     SUB,
		Routing:       0,
		ContentLength: uint32(len(receiveTopic)),
	}, &receiveTopic)
	if err != nil {
		panic(err)
	}

	i := 0
	bench := hrtime.NewBenchmark(runs)
	for bench.Next() {
		for q := 0; q < testSize; q++ {
			err := c.Write(&frisbee.Message{
				Id:            uint32(i),
				Operation:     PUB,
				Routing:       topicHash,
				ContentLength: uint32(len(data)),
			}, &data)
			if err != nil {
				panic(err)
			}
		}
		err := c.Write(&frisbee.Message{
			Id:            uint32(i),
			Operation:     PUB,
			Routing:       topicHash,
			ContentLength: uint32(len(END)),
		}, &END)
		if err != nil {
			panic(err)
		}
		i++
		<-complete
	}
	log.Println(bench.Histogram(10))

	err = c.Close()
	if err != nil {
		panic(err)
	}
}

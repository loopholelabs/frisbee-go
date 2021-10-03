/*
	Copyright 2021 Loophole Labs

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

		   http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package frisbee

import (
	"crypto/rand"
	"github.com/loopholelabs/frisbee/internal/protocol"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

func TestServerRaw(t *testing.T) {
	const testSize = 100
	const messageSize = 512
	addr := ":8192"
	clientRouter := make(ClientRouter)
	serverRouter := make(ServerRouter)

	serverIsRaw := make(chan struct{}, 1)

	serverRouter[protocol.MessagePing] = func(_ *Async, _ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		return
	}

	var rawServerConn, rawClientConn net.Conn
	serverRouter[protocol.MessagePacket] = func(c *Async, _ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		rawServerConn = c.Raw()
		serverIsRaw <- struct{}{}
		return
	}

	clientRouter[protocol.MessagePing] = func(_ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s := NewServer(addr, serverRouter, WithLogger(&emptyLogger))
	err := s.Start()
	require.NoError(t, err)

	c := NewClient(addr, clientRouter, WithLogger(&emptyLogger))
	_, err = c.Raw()
	assert.ErrorIs(t, ConnectionNotInitialized, err)

	err = c.Connect()
	require.NoError(t, err)

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	for q := 0; q < testSize; q++ {
		err := c.WriteMessage(&Message{
			To:            16,
			From:          32,
			Id:            uint64(q),
			Operation:     protocol.MessagePing,
			ContentLength: messageSize,
		}, &data)
		assert.NoError(t, err)
	}

	err = c.WriteMessage(&Message{
		To:            16,
		From:          32,
		Id:            0,
		Operation:     protocol.MessagePacket,
		ContentLength: 0,
	}, nil)
	assert.NoError(t, err)

	rawClientConn, err = c.Raw()
	require.NoError(t, err)

	<-serverIsRaw

	serverBytes := []byte("SERVER WRITE")

	write, err := rawServerConn.Write(serverBytes)
	assert.NoError(t, err)
	assert.Equal(t, len(serverBytes), write)

	clientBuffer := make([]byte, len(serverBytes))
	read, err := rawClientConn.Read(clientBuffer)
	assert.NoError(t, err)
	assert.Equal(t, len(serverBytes), read)

	assert.Equal(t, serverBytes, clientBuffer)

	err = c.Close()
	assert.NoError(t, err)
	err = rawClientConn.Close()
	assert.NoError(t, err)

	err = s.Shutdown()
	assert.NoError(t, err)
	err = rawServerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkThroughput(b *testing.B) {
	const testSize = 100000
	const messageSize = 512
	addr := ":8192"
	router := make(ServerRouter)

	router[protocol.MessagePing] = func(_ *Async, _ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	server := NewServer(addr, router, WithLogger(&emptyLogger))
	err := server.Start()
	if err != nil {
		log.Printf("Could not start server")
		panic(err)
	}

	frisbeeConn, err := ConnectAsync("tcp", addr, time.Minute*3, &emptyLogger, nil)
	if err != nil {
		log.Printf("Could not connect to server")
		panic(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				err := frisbeeConn.WriteMessage(&Message{
					To:            uint32(i),
					From:          uint32(i),
					Id:            uint64(q),
					Operation:     protocol.MessagePing,
					ContentLength: messageSize,
				}, &data)
				if err != nil {
					log.Printf("Could not write to server")
					panic(err)
				}
			}
		}
	})
	b.StopTimer()
	err = frisbeeConn.Close()
	if err != nil {
		log.Printf("Could not disconnect from server")
		panic(err)
	}
	err = server.Shutdown()
	if err != nil {
		log.Printf("Could not shut down server")
		panic(err)
	}
}

func BenchmarkThroughputWithResponse(b *testing.B) {
	const testSize = 100000
	const messageSize = 512
	addr := ":8192"
	router := make(ServerRouter)

	router[protocol.MessagePing] = func(_ *Async, incomingMessage Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		if incomingMessage.Id == testSize-1 {
			outgoingMessage = &Message{
				To:            16,
				From:          32,
				Id:            testSize,
				Operation:     protocol.MessagePong,
				ContentLength: 0,
			}
		}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	server := NewServer(addr, router, WithLogger(&emptyLogger))
	err := server.Start()
	if err != nil {
		log.Printf("Could not start server")
		panic(err)
	}

	frisbeeConn, err := ConnectAsync("tcp", addr, time.Minute*3, &emptyLogger, nil)
	if err != nil {
		log.Printf("Could not connect to server")
		panic(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				err := frisbeeConn.WriteMessage(&Message{
					To:            uint32(i),
					From:          uint32(i),
					Id:            uint64(q),
					Operation:     protocol.MessagePing,
					ContentLength: messageSize,
				}, &data)
				if err != nil {
					log.Printf("Could not write to server")
					panic(err)
				}
			}
			message, _, err := frisbeeConn.ReadMessage()
			if err != nil {
				panic(err)
			}

			if message.Id != testSize {
				log.Printf("Could retrieve data from server")
				panic("invalid decoded message id")
			}
		}

	})
	b.StopTimer()
	err = frisbeeConn.Close()
	if err != nil {
		log.Printf("Could not disconnect from server")
		panic(err)
	}
	err = server.Shutdown()
	if err != nil {
		log.Printf("Could not shut down server")
		panic(err)
	}
}

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
	"testing"
)

func TestClientRaw(t *testing.T) {
	t.Parallel()

	const testSize = 100
	const messageSize = 512
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
	s, err := NewServer(":0", serverRouter, WithLogger(&emptyLogger))
	require.NoError(t, err)

	err = s.Start()
	require.NoError(t, err)

	c, err := NewClient(s.listener.Addr().String(), clientRouter, WithLogger(&emptyLogger))
	assert.NoError(t, err)
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

	clientBytes := []byte("CLIENT WRITE")

	write, err := rawClientConn.Write(clientBytes)
	assert.NoError(t, err)
	assert.Equal(t, len(clientBytes), write)

	serverBuffer := make([]byte, len(clientBytes))
	read, err := rawServerConn.Read(serverBuffer)
	assert.NoError(t, err)
	assert.Equal(t, len(clientBytes), read)

	assert.Equal(t, clientBytes, serverBuffer)

	err = c.Close()
	assert.NoError(t, err)
	err = rawClientConn.Close()
	assert.NoError(t, err)

	err = s.Shutdown()
	assert.NoError(t, err)
	err = rawServerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkClientThroughput(b *testing.B) {
	const testSize = 100000
	const messageSize = 512

	clientRouter := make(ClientRouter)
	serverRouter := make(ServerRouter)

	serverRouter[protocol.MessagePing] = func(_ *Async, _ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		return
	}

	clientRouter[protocol.MessagePing] = func(_ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s, err := NewServer(":0", serverRouter, WithLogger(&emptyLogger))
	if err != nil {
		panic(err)
	}

	err = s.Start()
	if err != nil {
		panic(err)
	}

	c, err := NewClient(s.listener.Addr().String(), clientRouter, WithLogger(&emptyLogger))
	if err != nil {
		panic(err)
	}
	err = c.Connect()
	if err != nil {
		panic(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				err := c.WriteMessage(&Message{
					To:            uint32(i),
					From:          uint32(i),
					Id:            uint64(q),
					Operation:     protocol.MessagePing,
					ContentLength: messageSize,
				}, &data)
				if err != nil {
					panic(err)
				}
			}
		}
	})
	b.StopTimer()
	err = c.Close()
	if err != nil {
		panic(err)
	}
	err = s.Shutdown()
	if err != nil {
		panic(err)
	}
}

func BenchmarkClientThroughputResponse(b *testing.B) {
	const testSize = 100000
	const messageSize = 512
	clientRouter := make(ClientRouter)
	serverRouter := make(ServerRouter)

	finished := make(chan struct{}, 1)

	serverRouter[protocol.MessagePing] = func(_ *Async, incomingMessage Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
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

	clientRouter[protocol.MessagePong] = func(incomingMessage Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
		if incomingMessage.Id == testSize {
			finished <- struct{}{}
		}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s, err := NewServer(":0", serverRouter, WithLogger(&emptyLogger))
	if err != nil {
		panic(err)
	}

	err = s.Start()
	if err != nil {
		panic(err)
	}

	c, err := NewClient(s.listener.Addr().String(), clientRouter, WithLogger(&emptyLogger))
	if err != nil {
		panic(err)
	}
	err = c.Connect()
	if err != nil {
		panic(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	b.Run("test", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				err := c.WriteMessage(&Message{
					To:            uint32(i),
					From:          uint32(i),
					Id:            uint64(q),
					Operation:     protocol.MessagePing,
					ContentLength: messageSize,
				}, &data)
				if err != nil {
					panic(err)
				}
			}
			<-finished
		}
	})
	b.StopTimer()
	err = c.Close()
	if err != nil {
		panic(err)
	}
	err = s.Shutdown()
	if err != nil {
		panic(err)
	}
}

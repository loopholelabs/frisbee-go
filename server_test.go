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
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

func TestServerRaw(t *testing.T) {
	t.Parallel()

	const testSize = 100
	const messageSize = 512
	clientRouter := make(ClientRouter)
	serverRouter := make(ServerRouter)

	serverIsRaw := make(chan struct{}, 1)

	serverRouter[protocol.MessagePing] = func(_ *Async, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	var rawServerConn, rawClientConn net.Conn
	serverRouter[protocol.MessagePacket] = func(c *Async, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		rawServerConn = c.Raw()
		serverIsRaw <- struct{}{}
		return
	}

	clientRouter[protocol.MessagePing] = func(_ *packet.Packet) (outgoing *packet.Packet, action Action) {
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
	p := packet.Get()
	p.Write(data)
	p.Message.ContentLength = messageSize
	p.Message.Operation = protocol.MessagePing

	for q := 0; q < testSize; q++ {
		p.Message.Id = uint16(q)
		err = c.WriteMessage(p)
		assert.NoError(t, err)
	}

	p.Reset()
	p.Message.Operation = protocol.MessagePacket

	err = c.WriteMessage(p)
	require.NoError(t, err)

	packet.Put(p)

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
	const testSize = 1<<16 - 1
	const messageSize = 512

	router := make(ServerRouter)

	router[protocol.MessagePing] = func(_ *Async, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	server, err := NewServer(":0", router, WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	err = server.Start()
	if err != nil {
		b.Fatal(err)
	}

	frisbeeConn, err := ConnectAsync(server.listener.Addr().String(), time.Minute*3, &emptyLogger, nil)
	if err != nil {
		b.Fatal(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)
	p := packet.Get()
	p.Message.Operation = protocol.MessagePing

	p.Write(data)
	p.Message.ContentLength = messageSize

	b.Run("test", func(b *testing.B) {
		b.SetBytes(testSize * messageSize)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				p.Message.Id = uint16(q)
				err = frisbeeConn.WriteMessage(p)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
	b.StopTimer()

	packet.Put(p)

	err = frisbeeConn.Close()
	if err != nil {
		b.Fatal(err)
	}
	err = server.Shutdown()
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkThroughputWithResponse(b *testing.B) {
	const testSize = 1<<16 - 1
	const messageSize = 512

	router := make(ServerRouter)

	router[protocol.MessagePing] = func(_ *Async, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		if incoming.Message.Id == testSize-1 {
			incoming.Reset()
			incoming.Message.Id = testSize
			incoming.Message.Operation = protocol.MessagePong
			outgoing = incoming
		}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	server, err := NewServer(":0", router, WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	err = server.Start()
	if err != nil {
		b.Fatal(err)
	}

	frisbeeConn, err := ConnectAsync(server.listener.Addr().String(), time.Minute*3, &emptyLogger, nil)
	if err != nil {
		b.Fatal(err)
	}

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	p := packet.Get()
	p.Message.Operation = protocol.MessagePing

	p.Write(data)
	p.Message.ContentLength = messageSize

	b.Run("test", func(b *testing.B) {
		b.SetBytes(testSize * messageSize)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				p.Message.Id = uint16(q)
				err = frisbeeConn.WriteMessage(p)
				if err != nil {
					b.Fatal(err)
				}
			}
			readPacket, err := frisbeeConn.ReadMessage()
			if err != nil {
				b.Fatal(err)
			}

			if readPacket.Message.Id != testSize {
				b.Fatal("invalid decoded message id")
			}
			packet.Put(readPacket)
		}

	})
	b.StopTimer()

	packet.Put(p)

	err = frisbeeConn.Close()
	if err != nil {
		b.Fatal(err)
	}
	err = server.Shutdown()
	if err != nil {
		b.Fatal(err)
	}
}

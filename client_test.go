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
)

func TestClientRaw(t *testing.T) {
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
	p.Message.Operation = protocol.MessagePing
	p.Write(data)
	p.Message.ContentLength = messageSize

	for q := 0; q < testSize; q++ {
		p.Message.Id = uint16(q)
		err := c.WriteMessage(p)
		assert.NoError(t, err)
	}
	p.Reset()
	p.Message.Operation = protocol.MessagePacket

	err = c.WriteMessage(p)
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

	serverRouter[protocol.MessagePing] = func(_ *Async, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	clientRouter[protocol.MessagePing] = func(_ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s, err := NewServer(":0", serverRouter, WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	err = s.Start()
	if err != nil {
		b.Fatal(err)
	}

	c, err := NewClient(s.listener.Addr().String(), clientRouter, WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}
	err = c.Connect()
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
				err = c.WriteMessage(p)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
	b.StopTimer()
	packet.Put(p)

	err = c.Close()
	if err != nil {
		b.Fatal(err)
	}
	err = s.Shutdown()
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkClientThroughputResponse(b *testing.B) {
	const testSize = 1<<16 - 1
	const messageSize = 512

	clientRouter := make(ClientRouter)
	serverRouter := make(ServerRouter)

	finished := make(chan struct{}, 1)

	serverRouter[protocol.MessagePing] = func(_ *Async, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		if incoming.Message.Id == testSize-1 {
			incoming.Reset()
			incoming.Message.Id = testSize
			incoming.Message.Operation = protocol.MessagePong
			outgoing = incoming
		}
		return
	}

	clientRouter[protocol.MessagePong] = func(incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		if incoming.Message.Id == testSize {
			finished <- struct{}{}
		}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s, err := NewServer(":0", serverRouter, WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	err = s.Start()
	if err != nil {
		b.Fatal(err)
	}

	c, err := NewClient(s.listener.Addr().String(), clientRouter, WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}
	err = c.Connect()
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
				err = c.WriteMessage(p)
				if err != nil {
					b.Fatal(err)
				}
			}
			<-finished
		}
	})
	b.StopTimer()

	packet.Put(p)

	err = c.Close()
	if err != nil {
		b.Fatal(err)
	}
	err = s.Shutdown()
	if err != nil {
		b.Fatal(err)
	}
}

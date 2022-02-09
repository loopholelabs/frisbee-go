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
	"context"
	"crypto/rand"
	"github.com/loopholelabs/frisbee/internal/metadata"
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

// trunk-ignore-all(golangci-lint/staticcheck)

const (
	serverConnContextKey = "conn"
)

func TestServerRaw(t *testing.T) {
	t.Parallel()

	const testSize = 100
	const packetSize = 512
	clientHandlerTable := make(HandlerTable)
	serverHandlerTable := make(HandlerTable)

	serverIsRaw := make(chan struct{}, 1)

	serverHandlerTable[metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	var rawServerConn, rawClientConn net.Conn
	serverHandlerTable[metadata.PacketProbe] = func(ctx context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		conn := ctx.Value(serverConnContextKey).(*Async)
		rawServerConn = conn.Raw()
		serverIsRaw <- struct{}{}
		return
	}

	clientHandlerTable[metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s, err := NewServer(":0", serverHandlerTable, WithLogger(&emptyLogger))
	require.NoError(t, err)

	s.ConnContext = func(ctx context.Context, c *Async) context.Context {
		return context.WithValue(ctx, serverConnContextKey, c)
	}

	err = s.Start()
	require.NoError(t, err)

	c, err := NewClient(s.listener.Addr().String(), clientHandlerTable, context.Background(), WithLogger(&emptyLogger))
	assert.NoError(t, err)
	_, err = c.Raw()
	assert.ErrorIs(t, ConnectionNotInitialized, err)

	err = c.Connect()
	require.NoError(t, err)

	data := make([]byte, packetSize)
	_, _ = rand.Read(data)
	p := packet.Get()
	p.Write(data)
	p.Metadata.ContentLength = packetSize
	p.Metadata.Operation = metadata.PacketPing

	for q := 0; q < testSize; q++ {
		p.Metadata.Id = uint16(q)
		err = c.WritePacket(p)
		assert.NoError(t, err)
	}

	p.Reset()
	p.Metadata.Operation = metadata.PacketProbe

	err = c.WritePacket(p)
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
	const packetSize = 512

	handlerTable := make(HandlerTable)

	handlerTable[metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	server, err := NewServer(":0", handlerTable, WithLogger(&emptyLogger))
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

	data := make([]byte, packetSize)
	_, _ = rand.Read(data)
	p := packet.Get()
	p.Metadata.Operation = metadata.PacketPing

	p.Write(data)
	p.Metadata.ContentLength = packetSize

	b.Run("test", func(b *testing.B) {
		b.SetBytes(testSize * packetSize)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				p.Metadata.Id = uint16(q)
				err = frisbeeConn.WritePacket(p)
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
	const packetSize = 512

	handlerTable := make(HandlerTable)

	handlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		if incoming.Metadata.Id == testSize-1 {
			incoming.Reset()
			incoming.Metadata.Id = testSize
			incoming.Metadata.Operation = metadata.PacketPong
			outgoing = incoming
		}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	server, err := NewServer(":0", handlerTable, WithLogger(&emptyLogger))
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

	data := make([]byte, packetSize)
	_, _ = rand.Read(data)

	p := packet.Get()
	p.Metadata.Operation = metadata.PacketPing

	p.Write(data)
	p.Metadata.ContentLength = packetSize

	b.Run("test", func(b *testing.B) {
		b.SetBytes(testSize * packetSize)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				p.Metadata.Id = uint16(q)
				err = frisbeeConn.WritePacket(p)
				if err != nil {
					b.Fatal(err)
				}
			}
			readPacket, err := frisbeeConn.ReadPacket()
			if err != nil {
				b.Fatal(err)
			}

			if readPacket.Metadata.Id != testSize {
				b.Fatal("invalid decoded metadata id")
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

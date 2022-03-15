/*
	Copyright 2022 Loophole Labs

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
	"fmt"
	"github.com/loopholelabs/frisbee/pkg/metadata"
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
	"runtime"
	"sync"
	"testing"
)

// trunk-ignore-all(golangci-lint/staticcheck)

const (
	clientConnContextKey = "conn"
)

func TestClientRaw(t *testing.T) {
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
		conn := ctx.Value(clientConnContextKey).(*Async)
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
		return context.WithValue(ctx, clientConnContextKey, c)
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
	p.Metadata.Operation = metadata.PacketPing
	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize

	for q := 0; q < testSize; q++ {
		p.Metadata.Id = uint16(q)
		err := c.WritePacket(p)
		assert.NoError(t, err)
	}
	p.Reset()
	p.Metadata.Operation = metadata.PacketProbe

	err = c.WritePacket(p)
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

func TestClientStaleClose(t *testing.T) {
	t.Parallel()

	const testSize = 100
	const packetSize = 512

	clientHandlerTable := make(HandlerTable)
	serverHandlerTable := make(HandlerTable)

	finished := make(chan struct{}, 1)

	serverHandlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		if incoming.Metadata.Id == testSize-1 {
			outgoing = incoming
			action = CLOSE
		}
		return
	}

	clientHandlerTable[metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		finished <- struct{}{}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s, err := NewServer(":0", serverHandlerTable, WithLogger(&emptyLogger))
	require.NoError(t, err)

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
	p.Metadata.Operation = metadata.PacketPing
	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize

	for q := 0; q < testSize; q++ {
		p.Metadata.Id = uint16(q)
		err := c.WritePacket(p)
		assert.NoError(t, err)
	}
	packet.Put(p)
	<-finished

	_, err = c.conn.ReadPacket()
	assert.ErrorIs(t, err, ConnectionClosed)

	err = c.Close()
	assert.NoError(t, err)

	err = s.Shutdown()
	assert.NoError(t, err)
}

func BenchmarkThroughputClient(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	clientHandlerTable := make(HandlerTable)
	serverHandlerTable := make(HandlerTable)

	serverHandlerTable[metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	clientHandlerTable[metadata.PacketPong] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s, err := NewServer(":0", serverHandlerTable, WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	err = s.Start()
	if err != nil {
		b.Fatal(err)
	}

	c, err := NewClient(s.listener.Addr().String(), clientHandlerTable, context.Background(), WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}
	err = c.Connect()
	if err != nil {
		b.Fatal(err)
	}

	data := make([]byte, packetSize)
	_, _ = rand.Read(data)
	p := packet.Get()

	p.Metadata.Operation = metadata.PacketPing
	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize

	b.Run("test", func(b *testing.B) {
		b.SetBytes(testSize * packetSize)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				p.Metadata.Id = uint16(q)
				err = c.WritePacket(p)
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

func BenchmarkThroughputResponseClient(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	clientHandlerTable := make(HandlerTable)
	serverHandlerTable := make(HandlerTable)

	finished := make(chan struct{}, 1)

	serverHandlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		if incoming.Metadata.Id == testSize-1 {
			incoming.Reset()
			incoming.Metadata.Id = testSize
			incoming.Metadata.Operation = metadata.PacketPong
			outgoing = incoming
		}
		return
	}

	clientHandlerTable[metadata.PacketPong] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		if incoming.Metadata.Id == testSize {
			finished <- struct{}{}
		}
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s, err := NewServer(":0", serverHandlerTable, WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	err = s.Start()
	if err != nil {
		b.Fatal(err)
	}

	c, err := NewClient(s.listener.Addr().String(), clientHandlerTable, context.Background(), WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}
	err = c.Connect()
	if err != nil {
		b.Fatal(err)
	}

	data := make([]byte, packetSize)
	_, _ = rand.Read(data)
	p := packet.Get()
	p.Metadata.Operation = metadata.PacketPing

	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize

	b.Run("test", func(b *testing.B) {
		b.SetBytes(testSize * packetSize)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for q := 0; q < testSize; q++ {
				p.Metadata.Id = uint16(q)
				err = c.WritePacket(p)
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

func BenchmarkThroughputResponseMultipleClient(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	runner := func(b *testing.B, numClients int) {
		serverHandlerTable := make(HandlerTable)

		finished := make(chan struct{}, numClients)

		serverHandlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
			if incoming.Metadata.Id == testSize-1 {
				incoming.Reset()
				incoming.Metadata.Id = testSize
				incoming.Metadata.Operation = metadata.PacketPong
				outgoing = incoming
			}
			return
		}

		emptyLogger := zerolog.New(ioutil.Discard)
		s, err := NewServer(":0", serverHandlerTable, WithLogger(&emptyLogger))
		if err != nil {
			b.Fatal(err)
		}

		err = s.Start()
		if err != nil {
			b.Fatal(err)
		}

		data := make([]byte, packetSize)
		_, _ = rand.Read(data)
		p := packet.Get()
		p.Metadata.Operation = metadata.PacketPing

		p.Content.Write(data)
		p.Metadata.ContentLength = packetSize

		clients := make([]*Client, 0, numClients)

		for i := 0; i < numClients; i++ {
			clientHandlerTable := make(HandlerTable)
			clientHandlerTable[metadata.PacketPong] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
				if incoming.Metadata.Id == testSize {
					finished <- struct{}{}
				}
				return
			}
			c, err := NewClient(s.listener.Addr().String(), clientHandlerTable, context.Background(), WithLogger(&emptyLogger))
			if err != nil {
				b.Fatal(err)
			}
			err = c.Connect()
			if err != nil {
				b.Fatal(err)
			}
			clients = append(clients, c)
		}

		var wg sync.WaitGroup
		b.Run(fmt.Sprintf("Run with %d clients", numClients), func(b *testing.B) {
			b.SetBytes(testSize * packetSize)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				wg.Add(numClients)
				for i := 0; i < numClients; i++ {
					go func(i int) {
						for q := 0; q < testSize; q++ {
							p.Metadata.Id = uint16(q)
							err = clients[i].WritePacket(p)
							if err != nil {
								b.Error(err)
								wg.Done()
								return
							}
						}
						<-finished
						wg.Done()
					}(i)
				}
				wg.Wait()
			}

		})
		b.StopTimer()

		packet.Put(p)

		for i := 0; i < numClients; i++ {
			err = clients[i].Close()
			if err != nil {
				b.Fatal(err)
			}
		}

		err = s.Shutdown()
		if err != nil {
			b.Fatal(err)
		}
	}

	runner(b, 1)
	runner(b, 2)
	runner(b, 5)

	runner(b, runtime.NumCPU()/2)
	runner(b, runtime.NumCPU())

}

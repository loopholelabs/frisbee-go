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
	"github.com/loopholelabs/frisbee/pkg/metadata"
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/loopholelabs/testing/conn"
	"github.com/loopholelabs/testing/conn/pair"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
	"sync"
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
		c := ctx.Value(serverConnContextKey).(*Async)
		rawServerConn = c.Raw()
		serverIsRaw <- struct{}{}
		return
	}

	clientHandlerTable[metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	s, err := NewServer(serverHandlerTable, WithLogger(&emptyLogger))
	require.NoError(t, err)

	s.ConnContext = func(ctx context.Context, c *Async) context.Context {
		return context.WithValue(ctx, serverConnContextKey, c)
	}

	serverConn, clientConn, err := pair.New()
	require.NoError(t, err)

	go s.ServeConn(serverConn)

	c, err := NewClient(clientHandlerTable, context.Background(), WithLogger(&emptyLogger))
	assert.NoError(t, err)

	_, err = c.Raw()
	assert.ErrorIs(t, ConnectionNotInitialized, err)

	err = c.FromConn(clientConn)
	assert.NoError(t, err)

	data := make([]byte, packetSize)
	_, _ = rand.Read(data)
	p := packet.Get()
	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize
	p.Metadata.Operation = metadata.PacketPing
	assert.Equal(t, data, p.Content.B)

	for q := 0; q < testSize; q++ {
		p.Metadata.Id = uint16(q)
		err = c.WritePacket(p)
		assert.NoError(t, err)
	}

	p.Reset()
	assert.Equal(t, 0, len(p.Content.B))
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

func TestServerStaleClose(t *testing.T) {
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
	s, err := NewServer(serverHandlerTable, WithLogger(&emptyLogger))
	require.NoError(t, err)

	serverConn, clientConn, err := pair.New()
	require.NoError(t, err)

	go s.ServeConn(serverConn)

	c, err := NewClient(clientHandlerTable, context.Background(), WithLogger(&emptyLogger))
	assert.NoError(t, err)
	_, err = c.Raw()
	assert.ErrorIs(t, ConnectionNotInitialized, err)

	err = c.FromConn(clientConn)
	require.NoError(t, err)

	data := make([]byte, packetSize)
	_, _ = rand.Read(data)
	p := packet.Get()
	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize
	p.Metadata.Operation = metadata.PacketPing
	assert.Equal(t, data, p.Content.B)

	for q := 0; q < testSize; q++ {
		p.Metadata.Id = uint16(q)
		err = c.WritePacket(p)
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

func TestServerMultipleConnections(t *testing.T) {
	t.Parallel()

	const testSize = 100
	const packetSize = 512

	runner := func(t *testing.T, num int) {
		finished := make(chan struct{}, num)
		clientTables := make([]HandlerTable, num)
		for i := 0; i < num; i++ {
			clientTables[i] = make(HandlerTable)
			clientTables[i][metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
				finished <- struct{}{}
				return
			}
		}
		serverHandlerTable := make(HandlerTable)
		serverHandlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
			if incoming.Metadata.Id == testSize-1 {
				outgoing = incoming
				action = CLOSE
			}
			return
		}

		emptyLogger := zerolog.New(ioutil.Discard)
		s, err := NewServer(serverHandlerTable, WithLogger(&emptyLogger))
		require.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			err := s.Start(conn.Listen)
			require.NoError(t, err)
			wg.Done()
		}()

		time.Sleep(time.Millisecond * time.Duration(500*num))

		clients := make([]*Client, num)
		for i := 0; i < num; i++ {
			c, err := NewClient(clientTables[i], context.Background(), WithLogger(&emptyLogger))
			assert.NoError(t, err)
			_, err = c.Raw()
			assert.ErrorIs(t, ConnectionNotInitialized, err)

			err = c.Connect(s.listener.Addr().String())
			require.NoError(t, err)

			clients[i] = c
		}

		data := make([]byte, packetSize)
		_, err = rand.Read(data)
		assert.NoError(t, err)

		var clientWg sync.WaitGroup
		for i := 0; i < num; i++ {
			idx := i
			clientWg.Add(1)
			go func() {
				p := packet.Get()
				p.Content.Write(data)
				p.Metadata.ContentLength = packetSize
				p.Metadata.Operation = metadata.PacketPing
				assert.Equal(t, data, p.Content.B)
				for q := 0; q < testSize; q++ {
					p.Metadata.Id = uint16(q)
					err := clients[idx].WritePacket(p)
					assert.NoError(t, err)
				}
				<-finished
				err = clients[idx].Close()
				assert.NoError(t, err)
				clientWg.Done()
				packet.Put(p)
			}()
		}

		clientWg.Wait()

		err = s.Shutdown()
		assert.NoError(t, err)
		wg.Wait()

	}

	t.Run("1", func(t *testing.T) { runner(t, 1) })
	t.Run("2", func(t *testing.T) { runner(t, 2) })
	t.Run("3", func(t *testing.T) { runner(t, 3) })
	t.Run("5", func(t *testing.T) { runner(t, 5) })
	t.Run("10", func(t *testing.T) { runner(t, 10) })
	t.Run("100", func(t *testing.T) { runner(t, 100) })
}

func BenchmarkThroughputServer(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	handlerTable := make(HandlerTable)

	handlerTable[metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	emptyLogger := zerolog.New(ioutil.Discard)
	server, err := NewServer(handlerTable, WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	serverConn, clientConn, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	go server.ServeConn(serverConn)

	frisbeeConn := NewAsync(clientConn, &emptyLogger)

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

func BenchmarkThroughputResponseServer(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	serverConn, clientConn, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

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
	server, err := NewServer(handlerTable, WithLogger(&emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	go server.ServeConn(serverConn)

	frisbeeConn := NewAsync(clientConn, &emptyLogger)

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
				b.Fatal("invalid decoded metadata id", readPacket.Metadata.Id)
			}

			if readPacket.Metadata.Operation != metadata.PacketPong {
				b.Fatal("invalid decoded operation", readPacket.Metadata.Operation)
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

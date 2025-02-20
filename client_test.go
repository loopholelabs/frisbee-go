// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"context"
	"crypto/rand"
	"net"
	"testing"

	"github.com/loopholelabs/logging"
	"github.com/loopholelabs/testing/conn/pair"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/loopholelabs/frisbee-go/pkg/metadata"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
)

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

	emptyLogger := logging.Test(t, logging.Noop, t.Name())
	s, err := NewServer(serverHandlerTable, context.Background(), WithLogger(emptyLogger))
	require.NoError(t, err)

	s.SetConcurrency(1)

	s.ConnContext = func(ctx context.Context, c *Async) context.Context {
		return context.WithValue(ctx, clientConnContextKey, c)
	}

	serverConn, clientConn, err := pair.New()
	require.NoError(t, err)

	go s.ServeConn(serverConn)

	c, err := NewClient(clientHandlerTable, context.Background(), WithLogger(emptyLogger))
	assert.NoError(t, err)
	_, err = c.Raw()
	assert.ErrorIs(t, ConnectionNotInitialized, err)

	err = c.FromConn(clientConn)
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

	emptyLogger := logging.Test(t, logging.Noop, t.Name())
	s, err := NewServer(serverHandlerTable, context.Background(), WithLogger(emptyLogger))
	require.NoError(t, err)

	s.SetConcurrency(1)

	serverConn, clientConn, err := pair.New()
	require.NoError(t, err)

	go s.ServeConn(serverConn)

	c, err := NewClient(clientHandlerTable, context.Background(), WithLogger(emptyLogger))
	assert.NoError(t, err)
	_, err = c.Raw()
	assert.ErrorIs(t, ConnectionNotInitialized, err)

	err = c.FromConn(clientConn)
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
	DisableMaxContentLength(b)

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

	emptyLogger := logging.Test(b, logging.Noop, b.Name())
	s, err := NewServer(serverHandlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	s.SetConcurrency(1)

	serverConn, clientConn, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	go s.ServeConn(serverConn)

	c, err := NewClient(clientHandlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}
	err = c.FromConn(clientConn)
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
	DisableMaxContentLength(b)

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

	emptyLogger := logging.Test(b, logging.Noop, b.Name())
	s, err := NewServer(serverHandlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	s.SetConcurrency(1)

	serverConn, clientConn, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	go s.ServeConn(serverConn)

	c, err := NewClient(clientHandlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}
	err = c.FromConn(clientConn)
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

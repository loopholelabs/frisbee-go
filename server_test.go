// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/loopholelabs/logging"
	"github.com/loopholelabs/polyglot/v2"
	"github.com/loopholelabs/testing/conn"
	"github.com/loopholelabs/testing/conn/pair"

	"github.com/loopholelabs/frisbee-go/pkg/metadata"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
)

const (
	serverConnContextKey = "conn"
)

func TestServerRawSingle(t *testing.T) {
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

	emptyLogger := logging.Test(t, logging.Noop, t.Name())
	s, err := NewServer(serverHandlerTable, context.Background(), WithLogger(emptyLogger))
	require.NoError(t, err)

	s.SetConcurrency(1)

	s.ConnContext = func(ctx context.Context, c *Async) context.Context {
		return context.WithValue(ctx, serverConnContextKey, c)
	}

	serverConn, clientConn, err := pair.New()
	require.NoError(t, err)

	go s.ServeConn(serverConn)

	c, err := NewClient(clientHandlerTable, context.Background(), WithLogger(emptyLogger))
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
	expected := polyglot.NewBufferFromBytes(data)
	expected.MoveOffset(len(data))
	assert.Equal(t, expected.Bytes(), p.Content.Bytes())

	for q := 0; q < testSize; q++ {
		p.Metadata.Id = uint16(q)
		err = c.WritePacket(p)
		assert.NoError(t, err)
	}

	p.Reset()
	assert.Equal(t, 0, p.Content.Len())
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
	assert.Equal(t, cap(serverBytes), write)

	clientBuffer := make([]byte, cap(serverBytes))
	read, err := rawClientConn.Read(clientBuffer[:])
	assert.NoError(t, err)
	assert.Equal(t, cap(serverBytes), read)

	assert.Equal(t, serverBytes, clientBuffer[:read])

	err = c.Close()
	assert.NoError(t, err)
	err = rawClientConn.Close()
	assert.NoError(t, err)

	err = s.Shutdown()
	assert.NoError(t, err)
	err = rawServerConn.Close()
	assert.NoError(t, err)
}

func TestServerStaleCloseSingle(t *testing.T) {
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
	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize
	p.Metadata.Operation = metadata.PacketPing
	expected := polyglot.NewBufferFromBytes(data)
	expected.MoveOffset(len(data))
	assert.Equal(t, expected.Bytes(), p.Content.Bytes())

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

func TestServerMultipleConnectionsSingle(t *testing.T) {
	t.Parallel()

	const testSize = 100
	const packetSize = 512

	runner := func(t *testing.T, num int) {
		finished := make([]chan struct{}, num)
		clientTables := make([]HandlerTable, num)
		for i := 0; i < num; i++ {
			idx := i
			finished[idx] = make(chan struct{}, 1)
			clientTables[i] = make(HandlerTable)
			clientTables[i][metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
				finished[idx] <- struct{}{}
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

		emptyLogger := logging.Test(t, logging.Noop, t.Name())
		s, err := NewServer(serverHandlerTable, context.Background(), WithLogger(emptyLogger))
		require.NoError(t, err)

		s.SetConcurrency(1)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			err := s.Start(conn.Listen)
			require.NoError(t, err)
			wg.Done()
		}()

		<-s.started()
		listenAddr := s.listener.Addr().String()

		clients := make([]*Client, num)
		for i := 0; i < num; i++ {
			clients[i], err = NewClient(clientTables[i], context.Background(), WithLogger(emptyLogger))
			assert.NoError(t, err)
			_, err = clients[i].Raw()
			assert.ErrorIs(t, ConnectionNotInitialized, err)

			err = clients[i].Connect(listenAddr)
			require.NoError(t, err)
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
				expected := polyglot.NewBufferFromBytes(data)
				expected.MoveOffset(len(data))
				assert.Equal(t, expected.Bytes(), p.Content.Bytes())
				for q := 0; q < testSize; q++ {
					p.Metadata.Id = uint16(q)
					err := clients[idx].WritePacket(p)
					assert.NoError(t, err)
				}
				<-finished[idx]
				err := clients[idx].Close()
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

func TestServerRawUnlimited(t *testing.T) {
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

	emptyLogger := logging.Test(t, logging.Noop, t.Name())
	s, err := NewServer(serverHandlerTable, context.Background(), WithLogger(emptyLogger))
	require.NoError(t, err)

	s.SetConcurrency(0)

	s.ConnContext = func(ctx context.Context, c *Async) context.Context {
		return context.WithValue(ctx, serverConnContextKey, c)
	}

	serverConn, clientConn, err := pair.New()
	require.NoError(t, err)

	go s.ServeConn(serverConn)

	c, err := NewClient(clientHandlerTable, context.Background(), WithLogger(emptyLogger))
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
	expected := polyglot.NewBufferFromBytes(data)
	expected.MoveOffset(len(data))
	assert.Equal(t, expected.Bytes(), p.Content.Bytes())

	for q := 0; q < testSize; q++ {
		p.Metadata.Id = uint16(q)
		err = c.WritePacket(p)
		assert.NoError(t, err)
	}

	p.Reset()
	assert.Equal(t, 0, p.Content.Len())
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

func TestServerStaleCloseUnlimited(t *testing.T) {
	t.Parallel()

	const testSize = 100
	const packetSize = 512
	clientHandlerTable := make(HandlerTable)
	serverHandlerTable := make(HandlerTable)

	finished := make(chan struct{}, 1)

	var count atomic.Int32
	serverHandlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		if count.Add(1) == testSize-1 {
			outgoing = incoming
			action = CLOSE
			count.Store(0)
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

	s.SetConcurrency(0)

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
	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize
	p.Metadata.Operation = metadata.PacketPing
	expected := polyglot.NewBufferFromBytes(data)
	expected.MoveOffset(len(data))
	assert.Equal(t, expected.Bytes(), p.Content.Bytes())

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

func TestServerMultipleConnectionsUnlimited(t *testing.T) {
	t.Parallel()

	const testSize = 100
	const packetSize = 512

	runner := func(t *testing.T, num int) {
		finished := make([]chan struct{}, num)
		clientTables := make([]HandlerTable, num)
		for i := 0; i < num; i++ {
			idx := i
			finished[idx] = make(chan struct{}, 1)
			clientTables[i] = make(HandlerTable)
			clientTables[i][metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
				finished[idx] <- struct{}{}
				return
			}
		}
		clientCounts := make([]atomic.Uint32, num)

		serverHandlerTable := make(HandlerTable)
		serverHandlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
			if clientCounts[incoming.Metadata.Id].Add(1) == testSize-1 {
				outgoing = incoming
				action = CLOSE
				clientCounts[incoming.Metadata.Id].Store(0)
			}
			return
		}

		emptyLogger := logging.Test(t, logging.Noop, t.Name())
		s, err := NewServer(serverHandlerTable, context.Background(), WithLogger(emptyLogger))
		require.NoError(t, err)

		s.SetConcurrency(0)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			err := s.Start(conn.Listen)
			require.NoError(t, err)
			wg.Done()
		}()

		<-s.started()
		listenAddr := s.listener.Addr().String()

		clients := make([]*Client, num)
		for i := 0; i < num; i++ {
			clients[i], err = NewClient(clientTables[i], context.Background(), WithLogger(emptyLogger))
			assert.NoError(t, err)
			_, err = clients[i].Raw()
			assert.ErrorIs(t, ConnectionNotInitialized, err)

			err = clients[i].Connect(listenAddr)
			require.NoError(t, err)
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
				p.Metadata.Id = uint16(idx)
				expected := polyglot.NewBufferFromBytes(data)
				expected.MoveOffset(len(data))
				assert.Equal(t, expected.Bytes(), p.Content.Bytes())
				for q := 0; q < testSize; q++ {
					err := clients[idx].WritePacket(p)
					assert.NoError(t, err)
				}
				<-finished[idx]
				err := clients[idx].Close()
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

func TestServerRawLimited(t *testing.T) {
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

	emptyLogger := logging.Test(t, logging.Noop, t.Name())
	s, err := NewServer(serverHandlerTable, context.Background(), WithLogger(emptyLogger))
	require.NoError(t, err)

	s.SetConcurrency(10)

	s.ConnContext = func(ctx context.Context, c *Async) context.Context {
		return context.WithValue(ctx, serverConnContextKey, c)
	}

	serverConn, clientConn, err := pair.New()
	require.NoError(t, err)

	go s.ServeConn(serverConn)

	c, err := NewClient(clientHandlerTable, context.Background(), WithLogger(emptyLogger))
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
	expected := polyglot.NewBufferFromBytes(data)
	expected.MoveOffset(len(data))
	assert.Equal(t, expected.Bytes(), p.Content.Bytes())

	for q := 0; q < testSize; q++ {
		p.Metadata.Id = uint16(q)
		err = c.WritePacket(p)
		assert.NoError(t, err)
	}

	p.Reset()
	assert.Equal(t, 0, p.Content.Len())
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

func TestServerStaleCloseLimited(t *testing.T) {
	t.Parallel()

	const testSize = 100
	const packetSize = 512
	clientHandlerTable := make(HandlerTable)
	serverHandlerTable := make(HandlerTable)

	finished := make(chan struct{}, 1)

	var count atomic.Int32
	serverHandlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		if count.Add(1) == testSize-1 {
			outgoing = incoming
			action = CLOSE
			count.Store(0)
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

	s.SetConcurrency(10)

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
	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize
	p.Metadata.Operation = metadata.PacketPing
	expected := polyglot.NewBufferFromBytes(data)
	expected.MoveOffset(len(data))
	assert.Equal(t, expected.Bytes(), p.Content.Bytes())

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

func TestServerMultipleConnectionsLimited(t *testing.T) {
	t.Parallel()

	const testSize = 100
	const packetSize = 512

	runner := func(t *testing.T, num int) {
		finished := make([]chan struct{}, num)
		clientTables := make([]HandlerTable, num)
		for i := 0; i < num; i++ {
			idx := i
			finished[idx] = make(chan struct{}, 1)
			clientTables[i] = make(HandlerTable)
			clientTables[i][metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
				finished[idx] <- struct{}{}
				return
			}
		}

		clientCounts := make([]atomic.Uint32, num)

		serverHandlerTable := make(HandlerTable)
		serverHandlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
			if clientCounts[incoming.Metadata.Id].Add(1) == testSize-1 {
				outgoing = incoming
				action = CLOSE
				clientCounts[incoming.Metadata.Id].Store(0)
			}
			return
		}

		emptyLogger := logging.Test(t, logging.Noop, t.Name())
		s, err := NewServer(serverHandlerTable, context.Background(), WithLogger(emptyLogger))
		require.NoError(t, err)

		s.SetConcurrency(10)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			err := s.Start(conn.Listen)
			require.NoError(t, err)
			wg.Done()
		}()

		<-s.started()
		listenAddr := s.listener.Addr().String()

		clients := make([]*Client, num)
		for i := 0; i < num; i++ {
			clients[i], err = NewClient(clientTables[i], context.Background(), WithLogger(emptyLogger))
			assert.NoError(t, err)
			_, err = clients[i].Raw()
			assert.ErrorIs(t, ConnectionNotInitialized, err)

			err = clients[i].Connect(listenAddr)
			require.NoError(t, err)
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
				p.Metadata.Id = uint16(idx)
				expected := polyglot.NewBufferFromBytes(data)
				expected.MoveOffset(len(data))
				assert.Equal(t, expected.Bytes(), p.Content.Bytes())
				for q := 0; q < testSize; q++ {
					err := clients[idx].WritePacket(p)
					assert.NoError(t, err)
				}
				<-finished[idx]
				err := clients[idx].Close()
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

func TestServerInvalidPacket(t *testing.T) {
	t.Parallel()

	// Ensure request is rejected promptly.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)

	emptyLogger := logging.Test(t, logging.Noop, t.Name())
	s, err := NewServer(nil, context.Background(), WithLogger(emptyLogger))
	require.NoError(t, err)

	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go s.StartWithListener(ln)
	t.Cleanup(func() { s.Shutdown() })

	url := fmt.Sprintf("http://%s/", ln.Addr())
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	require.NoError(t, err)

	_, err = http.DefaultClient.Do(req)
	require.ErrorIs(t, err, io.EOF)
}

func BenchmarkThroughputServerSingle(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	handlerTable := make(HandlerTable)

	handlerTable[metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		return
	}

	emptyLogger := logging.Test(b, logging.Noop, b.Name())
	server, err := NewServer(handlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	server.SetConcurrency(1)

	serverConn, clientConn, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	go server.ServeConn(serverConn)

	frisbeeConn := NewAsync(clientConn, emptyLogger)

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

func BenchmarkThroughputServerUnlimited(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	handlerTable := make(HandlerTable)

	handlerTable[metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		time.Sleep(time.Millisecond * 50)
		return
	}

	emptyLogger := logging.Test(b, logging.Noop, b.Name())
	server, err := NewServer(handlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	server.SetConcurrency(0)

	serverConn, clientConn, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	go server.ServeConn(serverConn)

	frisbeeConn := NewAsync(clientConn, emptyLogger)

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

func BenchmarkThroughputServerLimited(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	handlerTable := make(HandlerTable)

	handlerTable[metadata.PacketPing] = func(_ context.Context, _ *packet.Packet) (outgoing *packet.Packet, action Action) {
		time.Sleep(time.Millisecond * 50)
		return
	}

	emptyLogger := logging.Test(b, logging.Noop, b.Name())
	server, err := NewServer(handlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	server.SetConcurrency(1 << 14)

	serverConn, clientConn, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	go server.ServeConn(serverConn)

	frisbeeConn := NewAsync(clientConn, emptyLogger)

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

func BenchmarkThroughputResponseServerSingle(b *testing.B) {
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

	emptyLogger := logging.Test(b, logging.Noop, b.Name())
	server, err := NewServer(handlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	server.SetConcurrency(1)

	go server.ServeConn(serverConn)

	frisbeeConn := NewAsync(clientConn, emptyLogger)

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

func BenchmarkThroughputResponseServerSlowSingle(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	serverConn, clientConn, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	handlerTable := make(HandlerTable)

	handlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		time.Sleep(time.Microsecond * 50)
		if incoming.Metadata.Id == testSize-1 {
			incoming.Reset()
			incoming.Metadata.Id = testSize
			incoming.Metadata.Operation = metadata.PacketPong
			outgoing = incoming
		}
		return
	}

	emptyLogger := logging.Test(b, logging.Noop, b.Name())
	server, err := NewServer(handlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	server.SetConcurrency(1)

	go server.ServeConn(serverConn)

	frisbeeConn := NewAsync(clientConn, emptyLogger)

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

func BenchmarkThroughputResponseServerSlowUnlimited(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	serverConn, clientConn, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	handlerTable := make(HandlerTable)

	var count atomic.Uint64
	handlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		time.Sleep(time.Microsecond * 50)
		if count.Add(1) == testSize-1 {
			incoming.Reset()
			incoming.Metadata.Id = testSize
			incoming.Metadata.Operation = metadata.PacketPong
			outgoing = incoming
			count.Store(0)
		}
		return
	}

	emptyLogger := logging.Test(b, logging.Noop, b.Name())
	server, err := NewServer(handlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	server.SetConcurrency(0)

	go server.ServeConn(serverConn)

	frisbeeConn := NewAsync(clientConn, emptyLogger)

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

func BenchmarkThroughputResponseServerSlowLimited(b *testing.B) {
	const testSize = 1<<16 - 1
	const packetSize = 512

	serverConn, clientConn, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	handlerTable := make(HandlerTable)

	var count atomic.Uint64
	handlerTable[metadata.PacketPing] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
		time.Sleep(time.Microsecond * 50)
		if count.Add(1) == testSize-1 {
			incoming.Reset()
			incoming.Metadata.Id = testSize
			incoming.Metadata.Operation = metadata.PacketPong
			outgoing = incoming
			count.Store(0)
		}
		return
	}

	emptyLogger := logging.Test(b, logging.Noop, b.Name())
	server, err := NewServer(handlerTable, context.Background(), WithLogger(emptyLogger))
	if err != nil {
		b.Fatal(err)
	}

	server.SetConcurrency(100)

	go server.ServeConn(serverConn)

	frisbeeConn := NewAsync(clientConn, emptyLogger)

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

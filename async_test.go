// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"crypto/rand"
	"io"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/loopholelabs/logging"
	"github.com/loopholelabs/polyglot/v2"
	"github.com/loopholelabs/testing/conn/pair"

	"github.com/loopholelabs/frisbee-go/pkg/packet"
)

func TestNewAsync(t *testing.T) {
	t.Parallel()

	const packetSize = 512

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32

	err := writerConn.WritePacket(p)
	require.NoError(t, err)
	packet.Put(p)

	p, err = readerConn.ReadPacket()
	require.NoError(t, err)
	require.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(64), p.Metadata.Id)
	assert.Equal(t, uint16(32), p.Metadata.Operation)
	assert.Equal(t, uint32(0), p.Metadata.ContentLength)
	assert.Equal(t, 0, p.Content.Len())

	data := make([]byte, packetSize)
	_, _ = rand.Read(data)

	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize

	err = writerConn.WritePacket(p)
	require.NoError(t, err)

	packet.Put(p)

	p, err = readerConn.ReadPacket()
	require.NoError(t, err)
	assert.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(64), p.Metadata.Id)
	assert.Equal(t, uint16(32), p.Metadata.Operation)
	assert.Equal(t, uint32(packetSize), p.Metadata.ContentLength)
	assert.Equal(t, len(data), p.Content.Len())
	expected := polyglot.NewBufferFromBytes(data)
	expected.MoveOffset(len(data))
	assert.Equal(t, expected.Bytes(), p.Content.Bytes())

	packet.Put(p)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestAsyncLargeWrite(t *testing.T) {
	t.Parallel()

	const testSize = 100000
	const packetSize = 512

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	randomData := make([][]byte, testSize)
	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Metadata.ContentLength = packetSize

	for i := 0; i < testSize; i++ {
		randomData[i] = make([]byte, packetSize)
		_, _ = rand.Read(randomData[i])
		p.Content.Write(randomData[i])
		err := writerConn.WritePacket(p)
		p.Content.Reset()
		assert.NoError(t, err)
	}
	packet.Put(p)

	for i := 0; i < testSize; i++ {
		p, err := readerConn.ReadPacket()
		assert.NoError(t, err)
		assert.NotNil(t, p.Metadata)
		assert.Equal(t, uint16(64), p.Metadata.Id)
		assert.Equal(t, uint16(32), p.Metadata.Operation)
		assert.Equal(t, uint32(packetSize), p.Metadata.ContentLength)
		assert.Equal(t, len(randomData[i]), p.Content.Len())
		expected := polyglot.NewBufferFromBytes(randomData[i])
		expected.MoveOffset(len(randomData[i]))
		assert.Equal(t, expected.Bytes(), p.Content.Bytes())
		packet.Put(p)
	}

	err := readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestAsyncRawConn(t *testing.T) {
	t.Parallel()

	const testSize = 100000
	const packetSize = 32

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	reader, writer, err := pair.New()
	require.NoError(t, err)

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	randomData := make([]byte, packetSize)
	_, _ = rand.Read(randomData)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Content.Write(randomData)
	p.Metadata.ContentLength = packetSize

	for i := 0; i < testSize; i++ {
		err := writerConn.WritePacket(p)
		assert.NoError(t, err)
	}

	packet.Put(p)

	for i := 0; i < testSize; i++ {
		p, err := readerConn.ReadPacket()
		assert.NoError(t, err)
		assert.NotNil(t, p.Metadata)
		assert.Equal(t, uint16(64), p.Metadata.Id)
		assert.Equal(t, uint16(32), p.Metadata.Operation)
		assert.Equal(t, uint32(packetSize), p.Metadata.ContentLength)
		assert.Equal(t, packetSize, p.Content.Len())
		expected := polyglot.NewBufferFromBytes(randomData)
		expected.MoveOffset(len(randomData))
		assert.Equal(t, expected.Bytes(), p.Content.Bytes())
	}

	rawReaderConn := readerConn.Raw()
	rawWriterConn := writerConn.Raw()

	rawWriteMessage := []byte("TEST CASE MESSAGE")

	written, err := rawReaderConn.Write(rawWriteMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), written)
	rawReadMessage := make([]byte, len(rawWriteMessage))
	read, err := rawWriterConn.Read(rawReadMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), read)
	assert.Equal(t, rawWriteMessage, rawReadMessage)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)

	assert.NoError(t, pair.Cleanup(rawReaderConn, rawWriterConn))
}

func TestAsyncReadClose(t *testing.T) {
	t.Parallel()

	reader, writer := net.Pipe()

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32

	err := writerConn.WritePacket(p)
	require.NoError(t, err)

	packet.Put(p)

	err = writerConn.Flush()
	require.NoError(t, err)

	p, err = readerConn.ReadPacket()
	require.NoError(t, err)
	require.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(64), p.Metadata.Id)
	assert.Equal(t, uint16(32), p.Metadata.Operation)
	assert.Equal(t, uint32(0), p.Metadata.ContentLength)
	assert.Equal(t, 0, p.Content.Len())

	err = readerConn.conn.Close()
	assert.NoError(t, err)

	time.Sleep(DefaultPingInterval * 2)

	err = writerConn.WritePacket(p)
	if err == nil {
		err = writerConn.Flush()
		assert.Error(t, err)
	}
	assert.Error(t, writerConn.Error())

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestAsyncReadAvailableClose(t *testing.T) {
	t.Parallel()

	reader, writer := net.Pipe()

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32

	err := writerConn.WritePacket(p)
	require.NoError(t, err)

	err = writerConn.WritePacket(p)
	require.NoError(t, err)

	packet.Put(p)

	err = writerConn.Close()
	require.NoError(t, err)

	p, err = readerConn.ReadPacket()
	require.NoError(t, err)
	require.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(64), p.Metadata.Id)
	assert.Equal(t, uint16(32), p.Metadata.Operation)
	assert.Equal(t, uint32(0), p.Metadata.ContentLength)
	assert.Equal(t, 0, p.Content.Len())

	p, err = readerConn.ReadPacket()
	require.NoError(t, err)
	require.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(64), p.Metadata.Id)
	assert.Equal(t, uint16(32), p.Metadata.Operation)
	assert.Equal(t, uint32(0), p.Metadata.ContentLength)
	assert.Equal(t, 0, p.Content.Len())

	_, err = readerConn.ReadPacket()
	require.Error(t, err)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestAsyncWriteClose(t *testing.T) {
	t.Parallel()

	reader, writer := net.Pipe()

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32

	err := writerConn.WritePacket(p)
	require.NoError(t, err)

	packet.Put(p)

	err = writerConn.Flush()
	assert.NoError(t, err)

	p, err = readerConn.ReadPacket()
	assert.NoError(t, err)
	assert.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(64), p.Metadata.Id)
	assert.Equal(t, uint16(32), p.Metadata.Operation)
	assert.Equal(t, uint32(0), p.Metadata.ContentLength)
	assert.Equal(t, 0, p.Content.Len())

	err = writerConn.WritePacket(p)
	assert.NoError(t, err)

	packet.Put(p)

	err = writerConn.conn.Close()
	assert.NoError(t, err)

	runtime.Gosched()
	time.Sleep(DefaultDeadline * 2)
	runtime.Gosched()

	_, err = readerConn.ReadPacket()
	assert.ErrorIs(t, err, ConnectionClosed)
	assert.ErrorIs(t, readerConn.Error(), io.EOF)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestAsyncTimeout(t *testing.T) {
	t.Parallel()

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	reader, writer, err := pair.New()
	require.NoError(t, err)

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32

	err = writerConn.WritePacket(p)
	assert.NoError(t, err)

	packet.Put(p)

	err = writerConn.Flush()
	assert.NoError(t, err)

	p, err = readerConn.ReadPacket()
	assert.NoError(t, err)
	assert.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(64), p.Metadata.Id)
	assert.Equal(t, uint16(32), p.Metadata.Operation)
	assert.Equal(t, uint32(0), p.Metadata.ContentLength)
	assert.Equal(t, 0, p.Content.Len())

	time.Sleep(DefaultDeadline * 2)

	err = writerConn.Error()
	require.NoError(t, err)

	err = writerConn.WritePacket(p)
	require.NoError(t, err)

	err = writerConn.Flush()
	require.NoError(t, err)

	packet.Put(p)

	time.Sleep(DefaultDeadline)
	require.Equal(t, 1, readerConn.incoming.Length())

	err = writerConn.conn.Close()
	require.NoError(t, err)

	runtime.Gosched()
	time.Sleep(DefaultDeadline * 2)
	runtime.Gosched()

	p, err = readerConn.ReadPacket()
	require.NoError(t, err)
	assert.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(64), p.Metadata.Id)
	assert.Equal(t, uint16(32), p.Metadata.Operation)
	assert.Equal(t, uint32(0), p.Metadata.ContentLength)
	assert.Equal(t, 0, p.Content.Len())

	_, err = readerConn.ReadPacket()
	require.ErrorIs(t, err, ConnectionClosed)

	err = readerConn.Error()
	if err == nil {
		runtime.Gosched()
		time.Sleep(DefaultDeadline * 3)
		runtime.Gosched()
	}
	require.Error(t, readerConn.Error())

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkAsyncThroughputPipe(b *testing.B) {
	const testSize = 100

	emptyLogger := logging.Test(b, logging.Noop, b.Name())

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	b.Run("32 Bytes", throughputRunner(testSize, 32, readerConn, writerConn))
	b.Run("512 Bytes", throughputRunner(testSize, 512, readerConn, writerConn))
	b.Run("1024 Bytes", throughputRunner(testSize, 1024, readerConn, writerConn))
	b.Run("2048 Bytes", throughputRunner(testSize, 2048, readerConn, writerConn))
	b.Run("4096 Bytes", throughputRunner(testSize, 4096, readerConn, writerConn))

	_ = readerConn.Close()
	_ = writerConn.Close()
}

func BenchmarkAsyncThroughputNetwork(b *testing.B) {
	const testSize = 100

	emptyLogger := logging.Test(b, logging.Noop, b.Name())

	reader, writer, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	b.Run("32 Bytes", throughputRunner(testSize, 32, readerConn, writerConn))
	b.Run("512 Bytes", throughputRunner(testSize, 512, readerConn, writerConn))
	b.Run("1024 Bytes", throughputRunner(testSize, 1024, readerConn, writerConn))
	b.Run("2048 Bytes", throughputRunner(testSize, 2048, readerConn, writerConn))
	b.Run("4096 Bytes", throughputRunner(testSize, 4096, readerConn, writerConn))

	_ = readerConn.Close()
	_ = writerConn.Close()
}

func BenchmarkAsyncThroughputNetworkMultiple(b *testing.B) {
	const testSize = 100

	throughputRunner := func(testSize uint32, packetSize uint32, readerConn Conn, writerConn Conn) func(b *testing.B) {
		return func(b *testing.B) {
			var err error

			randomData := make([]byte, packetSize)
			_, _ = rand.Read(randomData)

			p := packet.Get()
			p.Metadata.Id = 64
			p.Metadata.Operation = 32
			p.Content.Write(randomData)
			p.Metadata.ContentLength = packetSize

			for i := 0; i < b.N; i++ {
				done := make(chan struct{}, 1)
				errCh := make(chan error, 1)
				go func() {
					for i := uint32(0); i < testSize; i++ {
						p, err := readerConn.ReadPacket()
						if err != nil {
							errCh <- err
							return
						}
						packet.Put(p)
					}
					done <- struct{}{}
				}()
				for i := uint32(0); i < testSize; i++ {
					select {
					case err = <-errCh:
						b.Fatal(err)
					default:
						err = writerConn.WritePacket(p)
						if err != nil {
							b.Fatal(err)
						}
					}
				}
				select {
				case <-done:
					continue
				case err = <-errCh:
					b.Fatal(err)
				}
			}

			packet.Put(p)
		}
	}

	runner := func(numClients int, packetSize uint32) func(b *testing.B) {
		return func(b *testing.B) {
			var wg sync.WaitGroup
			wg.Add(numClients)
			b.SetBytes(int64(testSize * packetSize))
			b.ReportAllocs()
			for i := 0; i < numClients; i++ {
				go func() {
					emptyLogger := logging.Test(b, logging.Noop, b.Name())

					reader, writer, err := pair.New()
					if err != nil {
						b.Error(err)
					}

					readerConn := NewAsync(reader, emptyLogger)
					writerConn := NewAsync(writer, emptyLogger)
					throughputRunner(testSize, packetSize, readerConn, writerConn)(b)

					_ = readerConn.Close()
					_ = writerConn.Close()
					wg.Done()
				}()
			}
			wg.Wait()
		}
	}

	b.Run("1 Pair, 32 Bytes", runner(1, 32))
	b.Run("2 Pair, 32 Bytes", runner(2, 32))
	b.Run("5 Pair, 32 Bytes", runner(5, 32))
	b.Run("10 Pair, 32 Bytes", runner(10, 32))
	b.Run("Half CPU Pair, 32 Bytes", runner(runtime.NumCPU()/2, 32))
	b.Run("CPU Pair, 32 Bytes", runner(runtime.NumCPU(), 32))
	b.Run("Double CPU Pair, 32 Bytes", runner(runtime.NumCPU()*2, 32))

	b.Run("1 Pair, 512 Bytes", runner(1, 512))
	b.Run("2 Pair, 512 Bytes", runner(2, 512))
	b.Run("5 Pair, 512 Bytes", runner(5, 512))
	b.Run("10 Pair, 512 Bytes", runner(10, 512))
	b.Run("Half CPU Pair, 512 Bytes", runner(runtime.NumCPU()/2, 512))
	b.Run("CPU Pair, 512 Bytes", runner(runtime.NumCPU(), 512))
	b.Run("Double CPU Pair, 512 Bytes", runner(runtime.NumCPU()*2, 512))

	b.Run("1 Pair, 4096 Bytes", runner(1, 4096))
	b.Run("2 Pair, 4096 Bytes", runner(2, 4096))
	b.Run("5 Pair, 4096 Bytes", runner(5, 4096))
	b.Run("10 Pair, 4096 Bytes", runner(10, 4096))
	b.Run("Half CPU Pair, 4096 Bytes", runner(runtime.NumCPU()/2, 4096))
	b.Run("CPU Pair, 4096 Bytes", runner(runtime.NumCPU(), 4096))
	b.Run("Double CPU Pair, 4096 Bytes", runner(runtime.NumCPU()*2, 4096))
}

// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/loopholelabs/logging"
	"github.com/loopholelabs/polyglot/v2"
	"github.com/loopholelabs/testing/conn/pair"

	"github.com/loopholelabs/frisbee-go/pkg/metadata"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
)

func TestNewSync(t *testing.T) {
	t.Parallel()
	const packetSize = 512

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, emptyLogger)
	writerConn := NewSync(writer, emptyLogger)

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32

	go func() {
		start <- struct{}{}
		p, err := readerConn.ReadPacket()
		assert.NoError(t, err)
		assert.NotNil(t, p.Metadata)
		assert.Equal(t, uint16(64), p.Metadata.Id)
		assert.Equal(t, uint16(32), p.Metadata.Operation)
		assert.Equal(t, uint32(0), p.Metadata.ContentLength)
		assert.Equal(t, 0, p.Content.Len())
		end <- struct{}{}
		packet.Put(p)
	}()

	<-start
	err := writerConn.WritePacket(p)
	assert.NoError(t, err)
	<-end

	data := make([]byte, packetSize)
	_, _ = rand.Read(data)

	p.Content.Write(data)
	p.Metadata.ContentLength = packetSize

	go func() {
		start <- struct{}{}
		p, err := readerConn.ReadPacket()
		assert.NoError(t, err)
		assert.NotNil(t, p.Metadata)
		assert.Equal(t, uint16(64), p.Metadata.Id)
		assert.Equal(t, uint16(32), p.Metadata.Operation)
		assert.Equal(t, uint32(packetSize), p.Metadata.ContentLength)
		assert.Equal(t, packetSize, p.Content.Len())
		expected := polyglot.NewBufferFromBytes(data)
		expected.MoveOffset(len(data))
		assert.Equal(t, expected.Bytes(), p.Content.Bytes())
		end <- struct{}{}
		packet.Put(p)
	}()

	<-start
	err = writerConn.WritePacket(p)
	assert.NoError(t, err)

	packet.Put(p)
	<-end

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestSyncLargeWrite(t *testing.T) {
	t.Parallel()

	const testSize = 100000
	const packetSize = 512

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, emptyLogger)
	writerConn := NewSync(writer, emptyLogger)

	randomData := make([][]byte, testSize)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Metadata.ContentLength = packetSize

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	go func() {
		start <- struct{}{}
		for i := 0; i < testSize; i++ {
			p, err := readerConn.ReadPacket()
			assert.NoError(t, err)
			assert.NotNil(t, p.Metadata)
			assert.Equal(t, uint16(64), p.Metadata.Id)
			assert.Equal(t, uint16(32), p.Metadata.Operation)
			assert.Equal(t, uint32(packetSize), p.Metadata.ContentLength)
			assert.Equal(t, packetSize, p.Content.Len())
			expected := polyglot.NewBufferFromBytes(randomData[i])
			expected.MoveOffset(len(randomData[i]))
			assert.Equal(t, expected.Bytes(), p.Content.Bytes())
			packet.Put(p)
		}
		end <- struct{}{}
	}()

	<-start
	for i := 0; i < testSize; i++ {
		randomData[i] = make([]byte, packetSize)
		_, _ = rand.Read(randomData[i])
		p.Content.Write(randomData[i])
		err := writerConn.WritePacket(p)
		p.Content.Reset()
		assert.NoError(t, err)
	}
	<-end

	packet.Put(p)

	// Verify large writes past max length fails.
	big := make([]byte, 2*DefaultMaxContentLength)

	p = packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Metadata.ContentLength = uint32(len(big))
	p.Content.Write(big)

	err := writerConn.WritePacket(p)
	assert.ErrorIs(t, err, ContentLengthExceeded)
	p.Content.Reset()
	packet.Put(p)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestSyncLargeRead(t *testing.T) {
	t.Parallel()

	client, server, err := pair.New()
	require.NoError(t, err)

	serverConn := NewSync(server, logging.Test(t, logging.Noop, t.Name()))
	t.Cleanup(func() { serverConn.Close() })

	// Write a large packet that exceeds the maximum limit.
	bigData := make([]byte, 2*DefaultMaxContentLength+metadata.Size)
	binary.BigEndian.PutUint16(bigData[metadata.MagicOffset:metadata.MagicOffset+metadata.MagicSize], metadata.PacketMagicHeader)
	binary.BigEndian.PutUint16(bigData[metadata.IdOffset:metadata.IdOffset+metadata.IdSize], 0xFFFF)
	binary.BigEndian.PutUint16(bigData[metadata.OperationOffset:metadata.OperationOffset+metadata.OperationSize], 0xFFFF)
	binary.BigEndian.PutUint32(bigData[metadata.ContentLengthOffset:metadata.ContentLengthOffset+metadata.ContentLengthSize], 2*DefaultMaxContentLength)

	doneCh := make(chan any)
	go func() {
		defer close(doneCh)

		_, err = client.Write(bigData)

		// Verify client was disconnected.
		var opError *net.OpError
		assert.ErrorAs(t, err, &opError)
	}()

	// Read packet and very it failed.
	_, err = serverConn.ReadPacket()
	assert.ErrorIs(t, err, ContentLengthExceeded)

	<-doneCh
}

func TestSyncDisableMaxContentLength(t *testing.T) {
	// Don't run in parallel since it modifies DefaultMaxContentLength.

	oldMax := DefaultMaxContentLength
	DefaultMaxContentLength = 0
	t.Cleanup(func() { DefaultMaxContentLength = oldMax })

	logger := logging.Test(t, logging.Noop, t.Name())

	t.Run("read", func(t *testing.T) {
		client, server, err := pair.New()
		require.NoError(t, err)

		serverConn := NewSync(server, logger)
		t.Cleanup(func() { serverConn.Close() })

		// Write a large packet that would exceed the default maximum limit.
		content := make([]byte, 2*oldMax+metadata.Size)
		binary.BigEndian.PutUint16(content[metadata.MagicOffset:metadata.MagicOffset+metadata.MagicSize], metadata.PacketMagicHeader)
		binary.BigEndian.PutUint16(content[metadata.IdOffset:metadata.IdOffset+metadata.IdSize], 0xFFFF)
		binary.BigEndian.PutUint16(content[metadata.OperationOffset:metadata.OperationOffset+metadata.OperationSize], 0xFFFF)
		binary.BigEndian.PutUint32(content[metadata.ContentLengthOffset:metadata.ContentLengthOffset+metadata.ContentLengthSize], 2*oldMax)

		doneCh := make(chan any)
		go func() {
			defer close(doneCh)

			n, err := client.Write(content)
			assert.NoError(t, err)
			assert.Equal(t, int(2*oldMax+metadata.Size), n)
		}()

		// Verify packet can be read.
		p, err := serverConn.ReadPacket()
		assert.NoError(t, err)
		assert.Equal(t, int(2*oldMax), p.Content.Len())

		<-doneCh
	})

	t.Run("write", func(t *testing.T) {
		client, server, err := pair.New()
		require.NoError(t, err)

		clientConn := NewSync(client, logger)
		t.Cleanup(func() { clientConn.Close() })

		serverConn := NewSync(server, logger)
		t.Cleanup(func() { serverConn.Close() })

		doneCh := make(chan any)
		go func() {
			defer close(doneCh)

			p, err := serverConn.ReadPacket()
			assert.NoError(t, err)
			if err == nil {
				assert.Equal(t, int(2*oldMax), p.Content.Len())
			}
		}()

		p := packet.Get()
		t.Cleanup(func() {
			p.Content.Reset()
			packet.Put(p)
		})

		// Write a large packet that would exceed the default maximum limit.
		content := make([]byte, 2*oldMax)

		p.Metadata.Id = 64
		p.Metadata.Operation = 32
		p.Metadata.ContentLength = uint32(len(content))
		p.Content.Write(content)

		err = clientConn.WritePacket(p)
		require.NoError(t, err)

		<-doneCh
	})
}

func TestSyncRawConn(t *testing.T) {
	t.Parallel()

	const testSize = 100000
	const packetSize = 32

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	reader, writer, err := pair.New()
	require.NoError(t, err)

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	readerConn := NewSync(reader, emptyLogger)
	writerConn := NewSync(writer, emptyLogger)

	randomData := make([]byte, packetSize)
	_, _ = rand.Read(randomData)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Content.Write(randomData)
	p.Metadata.ContentLength = packetSize

	go func() {
		start <- struct{}{}
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
			packet.Put(p)
		}
		end <- struct{}{}
	}()

	<-start
	for i := 0; i < testSize; i++ {
		err := writerConn.WritePacket(p)
		assert.NoError(t, err)
	}
	<-end

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

	err = rawReaderConn.Close()
	assert.NoError(t, err)
	err = rawWriterConn.Close()
	assert.NoError(t, err)
}

func TestSyncInvalid(t *testing.T) {
	t.Parallel()

	client, server, err := pair.New()
	require.NoError(t, err)

	serverConn := NewSync(server, logging.Test(t, logging.Noop, t.Name()))
	t.Cleanup(func() { serverConn.Close() })

	httpReq := []byte(`GET / HTTP/1.1
Host: www.example.com
User-Agent: curl/8.9.1
Accept: */*`)
	_, err = client.Write(httpReq)
	assert.NoError(t, err)

	_, err = serverConn.ReadPacket()
	assert.ErrorIs(t, err, InvalidMagicHeader)
	assert.ErrorIs(t, serverConn.Error(), InvalidMagicHeader)
}

func TestSyncReadClose(t *testing.T) {
	t.Parallel()

	reader, writer := net.Pipe()

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	readerConn := NewSync(reader, emptyLogger)
	writerConn := NewSync(writer, emptyLogger)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	go func() {
		start <- struct{}{}
		p, err := readerConn.ReadPacket()
		assert.NoError(t, err)
		assert.NotNil(t, p.Metadata)
		assert.Equal(t, uint16(64), p.Metadata.Id)
		assert.Equal(t, uint16(32), p.Metadata.Operation)
		assert.Equal(t, uint32(0), p.Metadata.ContentLength)
		assert.Equal(t, 0, p.Content.Len())
		end <- struct{}{}
		packet.Put(p)
	}()

	<-start
	err := writerConn.WritePacket(p)
	assert.NoError(t, err)
	<-end

	err = readerConn.conn.Close()
	assert.NoError(t, err)

	err = writerConn.WritePacket(p)
	assert.Error(t, err)
	assert.ErrorIs(t, writerConn.Error(), io.ErrClosedPipe)

	packet.Put(p)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestSyncWriteClose(t *testing.T) {
	t.Parallel()

	reader, writer := net.Pipe()

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	readerConn := NewSync(reader, emptyLogger)
	writerConn := NewSync(writer, emptyLogger)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	go func() {
		start <- struct{}{}
		p, err := readerConn.ReadPacket()
		assert.NoError(t, err)
		assert.NotNil(t, p.Metadata)
		assert.Equal(t, uint16(64), p.Metadata.Id)
		assert.Equal(t, uint16(32), p.Metadata.Operation)
		assert.Equal(t, uint32(0), p.Metadata.ContentLength)
		assert.Equal(t, 0, p.Content.Len())
		packet.Put(p)
		end <- struct{}{}
	}()

	<-start
	err := writerConn.WritePacket(p)
	assert.NoError(t, err)
	<-end

	packet.Put(p)

	err = writerConn.conn.Close()
	assert.NoError(t, err)

	_, err = readerConn.ReadPacket()
	assert.ErrorIs(t, err, io.EOF)
	assert.ErrorIs(t, readerConn.Error(), io.EOF)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkSyncThroughputPipe(b *testing.B) {
	const testSize = 100

	emptyLogger := logging.Test(b, logging.Noop, b.Name())

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, emptyLogger)
	writerConn := NewSync(writer, emptyLogger)

	b.Run("32 Bytes", throughputRunner(testSize, 32, readerConn, writerConn))
	b.Run("512 Bytes", throughputRunner(testSize, 512, readerConn, writerConn))
	b.Run("1024 Bytes", throughputRunner(testSize, 1024, readerConn, writerConn))
	b.Run("2048 Bytes", throughputRunner(testSize, 2048, readerConn, writerConn))
	b.Run("4096 Bytes", throughputRunner(testSize, 4096, readerConn, writerConn))

	_ = readerConn.Close()
	_ = writerConn.Close()
}

func BenchmarkSyncThroughputNetwork(b *testing.B) {
	const testSize = 100

	emptyLogger := logging.Test(b, logging.Noop, b.Name())

	reader, writer, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	readerConn := NewSync(reader, emptyLogger)
	writerConn := NewSync(writer, emptyLogger)

	b.Run("32 Bytes", throughputRunner(testSize, 32, readerConn, writerConn))
	b.Run("512 Bytes", throughputRunner(testSize, 512, readerConn, writerConn))
	b.Run("1024 Bytes", throughputRunner(testSize, 1024, readerConn, writerConn))
	b.Run("2048 Bytes", throughputRunner(testSize, 2048, readerConn, writerConn))
	b.Run("4096 Bytes", throughputRunner(testSize, 4096, readerConn, writerConn))

	_ = readerConn.Close()
	_ = writerConn.Close()
}

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
	"github.com/loopholelabs/frisbee/internal/queue"
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/loopholelabs/testing/conn/pair"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

func TestNewAsync(t *testing.T) {
	t.Parallel()

	const packetSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger, queue.NewBounded(DefaultBufferSize))
	writerConn := NewAsync(writer, &emptyLogger, queue.NewBounded(DefaultBufferSize))

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
	assert.Equal(t, 0, len(p.Content))

	data := make([]byte, packetSize)
	_, _ = rand.Read(data)

	p.Write(data)
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
	assert.Equal(t, len(data), len(p.Content))
	assert.Equal(t, data, p.Content)

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

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger, queue.NewBounded(DefaultBufferSize))
	writerConn := NewAsync(writer, &emptyLogger, queue.NewBounded(DefaultBufferSize))

	randomData := make([][]byte, testSize)
	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Metadata.ContentLength = packetSize

	for i := 0; i < testSize; i++ {
		randomData[i] = make([]byte, packetSize)
		_, _ = rand.Read(randomData[i])
		p.Write(randomData[i])
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
		assert.Equal(t, len(randomData[i]), len(p.Content))
		assert.Equal(t, randomData[i], p.Content)
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

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	require.NoError(t, err)

	readerConn := NewAsync(reader, &emptyLogger, queue.NewBounded(DefaultBufferSize))
	writerConn := NewAsync(writer, &emptyLogger, queue.NewBounded(DefaultBufferSize))

	randomData := make([]byte, packetSize)
	_, _ = rand.Read(randomData)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Write(randomData)
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
		assert.Equal(t, len(randomData), len(p.Content))
		assert.Equal(t, randomData, p.Content)
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

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := NewAsync(reader, &emptyLogger, queue.NewBounded(DefaultBufferSize))
	writerConn := NewAsync(writer, &emptyLogger, queue.NewBounded(DefaultBufferSize))

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
	assert.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(64), p.Metadata.Id)
	assert.Equal(t, uint16(32), p.Metadata.Operation)
	assert.Equal(t, uint32(0), p.Metadata.ContentLength)
	assert.Equal(t, 0, len(p.Content))

	err = readerConn.conn.Close()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 100)

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

func TestAsyncWriteClose(t *testing.T) {
	t.Parallel()

	reader, writer := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := NewAsync(reader, &emptyLogger, queue.NewBounded(DefaultBufferSize))
	writerConn := NewAsync(writer, &emptyLogger, queue.NewBounded(DefaultBufferSize))

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
	assert.Equal(t, 0, len(p.Content))

	err = writerConn.WritePacket(p)
	assert.NoError(t, err)

	packet.Put(p)

	err = writerConn.conn.Close()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 50)

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

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	require.NoError(t, err)

	readerConn := NewAsync(reader, &emptyLogger, queue.NewBounded(DefaultBufferSize))
	writerConn := NewAsync(writer, &emptyLogger, queue.NewBounded(DefaultBufferSize))

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
	assert.Equal(t, 0, len(p.Content))

	time.Sleep(defaultDeadline * 5)

	err = writerConn.Error()
	assert.NoError(t, err)

	err = writerConn.WritePacket(p)
	assert.NoError(t, err)

	packet.Put(p)

	err = writerConn.conn.Close()
	assert.NoError(t, err)

	time.Sleep(defaultDeadline * 5)

	_, err = readerConn.ReadPacket()
	assert.ErrorIs(t, err, ConnectionClosed)
	assert.Error(t, readerConn.Error())

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkAsyncThroughputPipe(b *testing.B) {
	const testSize = 100

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger, queue.NewBounded(DefaultBufferSize))
	writerConn := NewAsync(writer, &emptyLogger, queue.NewBounded(DefaultBufferSize))

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

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	readerConn := NewAsync(reader, &emptyLogger, queue.NewBounded(DefaultBufferSize))
	writerConn := NewAsync(writer, &emptyLogger, queue.NewBounded(DefaultBufferSize))

	b.Run("32 Bytes", throughputRunner(testSize, 32, readerConn, writerConn))
	b.Run("512 Bytes", throughputRunner(testSize, 512, readerConn, writerConn))
	b.Run("1024 Bytes", throughputRunner(testSize, 1024, readerConn, writerConn))
	b.Run("2048 Bytes", throughputRunner(testSize, 2048, readerConn, writerConn))
	b.Run("4096 Bytes", throughputRunner(testSize, 4096, readerConn, writerConn))

	_ = readerConn.Close()
	_ = writerConn.Close()
}

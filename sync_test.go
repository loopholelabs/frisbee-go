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
	"crypto/rand"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"github.com/loopholelabs/polyglot-go"
	"github.com/loopholelabs/testing/conn/pair"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net"
	"testing"
)

func TestNewSync(t *testing.T) {
	t.Parallel()
	const packetSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

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
		assert.Equal(t, 0, len(*p.Content))
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
		assert.Equal(t, packetSize, len(*p.Content))
		assert.Equal(t, polyglot.Buffer(data), *p.Content)
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

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

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
			assert.Equal(t, packetSize, len(*p.Content))
			assert.Equal(t, polyglot.Buffer(randomData[i]), *p.Content)
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

	err := readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestSyncRawConn(t *testing.T) {
	t.Parallel()

	const testSize = 100000
	const packetSize = 32

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	require.NoError(t, err)

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

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
			assert.Equal(t, packetSize, len(*p.Content))
			assert.Equal(t, polyglot.Buffer(randomData), *p.Content)
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

func TestSyncReadClose(t *testing.T) {
	t.Parallel()

	reader, writer := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

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
		assert.Equal(t, 0, len(*p.Content))
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

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

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
		assert.Equal(t, 0, len(*p.Content))
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

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

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

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	b.Run("32 Bytes", throughputRunner(testSize, 32, readerConn, writerConn))
	b.Run("512 Bytes", throughputRunner(testSize, 512, readerConn, writerConn))
	b.Run("1024 Bytes", throughputRunner(testSize, 1024, readerConn, writerConn))
	b.Run("2048 Bytes", throughputRunner(testSize, 2048, readerConn, writerConn))
	b.Run("4096 Bytes", throughputRunner(testSize, 4096, readerConn, writerConn))

	_ = readerConn.Close()
	_ = writerConn.Close()
}

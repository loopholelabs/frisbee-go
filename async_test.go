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

	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	p := packet.Get()
	p.Message.Id = 64
	p.Message.Operation = 32

	err := writerConn.WriteMessage(p)
	require.NoError(t, err)
	packet.Put(p)

	p, err = readerConn.ReadMessage()
	require.NoError(t, err)
	require.NotNil(t, p.Message)
	assert.Equal(t, uint16(64), p.Message.Id)
	assert.Equal(t, uint16(32), p.Message.Operation)
	assert.Equal(t, uint32(0), p.Message.ContentLength)
	assert.Equal(t, 0, len(p.Content))

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	p.Write(data)
	p.Message.ContentLength = messageSize

	err = writerConn.WriteMessage(p)
	require.NoError(t, err)

	packet.Put(p)

	p, err = readerConn.ReadMessage()
	require.NoError(t, err)
	assert.NotNil(t, p.Message)
	assert.Equal(t, uint16(64), p.Message.Id)
	assert.Equal(t, uint16(32), p.Message.Operation)
	assert.Equal(t, uint32(messageSize), p.Message.ContentLength)
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
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	randomData := make([][]byte, testSize)
	p := packet.Get()
	p.Message.Id = 64
	p.Message.Operation = 32
	p.Message.ContentLength = messageSize

	for i := 0; i < testSize; i++ {
		randomData[i] = make([]byte, messageSize)
		_, _ = rand.Read(randomData[i])
		p.Write(randomData[i])
		err := writerConn.WriteMessage(p)
		assert.NoError(t, err)
	}
	packet.Put(p)

	for i := 0; i < testSize; i++ {
		p, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.NotNil(t, p.Message)
		assert.Equal(t, uint16(64), p.Message.Id)
		assert.Equal(t, uint16(32), p.Message.Operation)
		assert.Equal(t, uint32(messageSize), p.Message.ContentLength)
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
	const messageSize = 32

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	require.NoError(t, err)

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)
	_, _ = rand.Read(randomData)

	p := packet.Get()
	p.Message.Id = 64
	p.Message.Operation = 32
	p.Write(randomData)
	p.Message.ContentLength = messageSize

	for i := 0; i < testSize; i++ {
		err := writerConn.WriteMessage(p)
		assert.NoError(t, err)
	}

	packet.Put(p)

	for i := 0; i < testSize; i++ {
		p, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.NotNil(t, p.Message)
		assert.Equal(t, uint16(64), p.Message.Id)
		assert.Equal(t, uint16(32), p.Message.Operation)
		assert.Equal(t, uint32(messageSize), p.Message.ContentLength)
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

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	p := packet.Get()
	p.Message.Id = 64
	p.Message.Operation = 32

	err := writerConn.WriteMessage(p)
	require.NoError(t, err)

	packet.Put(p)

	err = writerConn.Flush()
	require.NoError(t, err)

	p, err = readerConn.ReadMessage()
	require.NoError(t, err)
	assert.NotNil(t, p.Message)
	assert.Equal(t, uint16(64), p.Message.Id)
	assert.Equal(t, uint16(32), p.Message.Operation)
	assert.Equal(t, uint32(0), p.Message.ContentLength)
	assert.Equal(t, 0, len(p.Content))

	err = readerConn.conn.Close()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 100)

	err = writerConn.WriteMessage(p)
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

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	p := packet.Get()
	p.Message.Id = 64
	p.Message.Operation = 32

	err := writerConn.WriteMessage(p)
	require.NoError(t, err)

	packet.Put(p)

	err = writerConn.Flush()
	assert.NoError(t, err)

	p, err = readerConn.ReadMessage()
	assert.NoError(t, err)
	assert.NotNil(t, p.Message)
	assert.Equal(t, uint16(64), p.Message.Id)
	assert.Equal(t, uint16(32), p.Message.Operation)
	assert.Equal(t, uint32(0), p.Message.ContentLength)
	assert.Equal(t, 0, len(p.Content))

	err = writerConn.WriteMessage(p)
	assert.NoError(t, err)

	packet.Put(p)

	err = writerConn.conn.Close()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 50)

	_, err = readerConn.ReadMessage()
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

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	p := packet.Get()
	p.Message.Id = 64
	p.Message.Operation = 32

	err = writerConn.WriteMessage(p)
	assert.NoError(t, err)

	packet.Put(p)

	err = writerConn.Flush()
	assert.NoError(t, err)

	p, err = readerConn.ReadMessage()
	assert.NoError(t, err)
	assert.NotNil(t, p.Message)
	assert.Equal(t, uint16(64), p.Message.Id)
	assert.Equal(t, uint16(32), p.Message.Operation)
	assert.Equal(t, uint32(0), p.Message.ContentLength)
	assert.Equal(t, 0, len(p.Content))

	time.Sleep(defaultDeadline * 5)

	err = writerConn.Error()
	assert.NoError(t, err)

	err = writerConn.WriteMessage(p)
	assert.NoError(t, err)

	packet.Put(p)

	err = writerConn.conn.Close()
	assert.NoError(t, err)

	time.Sleep(defaultDeadline * 5)

	_, err = readerConn.ReadMessage()
	assert.ErrorIs(t, err, ConnectionClosed)
	assert.Error(t, readerConn.Error())

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkAsyncThroughputPipe(b *testing.B) {
	const testSize = 100000

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	throughputRunner := func(messageSize uint32) func(b *testing.B) {
		return func(b *testing.B) {
			b.SetBytes(testSize * int64(messageSize))
			b.ReportAllocs()

			randomData := make([]byte, messageSize)

			p := packet.Get()
			p.Message.Id = 64
			p.Message.Operation = 32
			p.Write(randomData)
			p.Message.ContentLength = messageSize

			done := make(chan struct{}, 1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				go func() {
					for i := 0; i < testSize; i++ {
						p, _ := readerConn.ReadMessage()
						packet.Put(p)
					}
					done <- struct{}{}
				}()
				for i := 0; i < testSize; i++ {
					_ = writerConn.WriteMessage(p)
				}
				<-done
			}
			b.StopTimer()

			packet.Put(p)
		}
	}

	b.Run("32 Bytes", throughputRunner(32))
	b.Run("512 Bytes", throughputRunner(512))
	b.Run("1024 Bytes", throughputRunner(1024))
	b.Run("2048 Bytes", throughputRunner(2048))
	b.Run("4096 Bytes", throughputRunner(4096))

	_ = readerConn.Close()
	_ = writerConn.Close()
}

func BenchmarkAsyncThroughputNetwork(b *testing.B) {
	const testSize = 100000

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, _ := pair.New()

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	throughputRunner := func(messageSize uint32) func(b *testing.B) {
		return func(b *testing.B) {
			b.SetBytes(testSize * int64(messageSize))
			b.ReportAllocs()

			randomData := make([]byte, messageSize)

			p := packet.Get()
			p.Message.Id = 64
			p.Message.Operation = 32
			p.Write(randomData)
			p.Message.ContentLength = messageSize

			done := make(chan struct{}, 1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				go func() {
					for i := 0; i < testSize; i++ {
						p, _ := readerConn.ReadMessage()
						packet.Put(p)
					}
					done <- struct{}{}
				}()
				for i := 0; i < testSize; i++ {
					_ = writerConn.WriteMessage(p)
				}
				<-done
			}
			b.StopTimer()

			packet.Put(p)
		}
	}
	b.Run("32 Bytes", throughputRunner(32))
	b.Run("512 Bytes", throughputRunner(512))
	b.Run("1024 Bytes", throughputRunner(1024))
	b.Run("2048 Bytes", throughputRunner(2048))
	b.Run("4096 Bytes", throughputRunner(4096))
	b.Run("1mb", throughputRunner(1<<20))

	_ = readerConn.Close()
	_ = writerConn.Close()
}

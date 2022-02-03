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
)

func TestNewSync(t *testing.T) {
	t.Parallel()
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	p := packet.Get()
	p.Message.Id = 64
	p.Message.Operation = 32

	go func() {
		start <- struct{}{}
		p, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.NotNil(t, p.Message)
		assert.Equal(t, uint16(64), p.Message.Id)
		assert.Equal(t, uint16(32), p.Message.Operation)
		assert.Equal(t, uint32(0), p.Message.ContentLength)
		assert.Equal(t, 0, len(p.Content))
		end <- struct{}{}
		packet.Put(p)
	}()

	<-start
	err := writerConn.WriteMessage(p)
	assert.NoError(t, err)
	<-end

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	p.Write(data)
	p.Message.ContentLength = messageSize

	go func() {
		start <- struct{}{}
		p, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.NotNil(t, p.Message)
		assert.Equal(t, uint16(64), p.Message.Id)
		assert.Equal(t, uint16(32), p.Message.Operation)
		assert.Equal(t, uint32(messageSize), p.Message.ContentLength)
		assert.Equal(t, messageSize, len(p.Content))
		assert.Equal(t, data, p.Content)
		end <- struct{}{}
		packet.Put(p)
	}()

	<-start
	err = writerConn.WriteMessage(p)
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
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([][]byte, testSize)

	p := packet.Get()
	p.Message.Id = 64
	p.Message.Operation = 32
	p.Message.ContentLength = messageSize

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	go func() {
		start <- struct{}{}
		for i := 0; i < testSize; i++ {
			p, err := readerConn.ReadMessage()
			assert.NoError(t, err)
			assert.NotNil(t, p.Message)
			assert.Equal(t, uint16(64), p.Message.Id)
			assert.Equal(t, uint16(32), p.Message.Operation)
			assert.Equal(t, uint32(messageSize), p.Message.ContentLength)
			assert.Equal(t, messageSize, len(p.Content))
			assert.Equal(t, randomData[i], p.Content)
			packet.Put(p)
		}
		end <- struct{}{}
	}()

	<-start
	for i := 0; i < testSize; i++ {
		randomData[i] = make([]byte, messageSize)
		_, _ = rand.Read(randomData[i])
		p.Write(randomData[i])
		err := writerConn.WriteMessage(p)
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
	const messageSize = 32

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	require.NoError(t, err)

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)
	_, _ = rand.Read(randomData)

	p := packet.Get()
	p.Message.Id = 64
	p.Message.Operation = 32
	p.Write(randomData)
	p.Message.ContentLength = messageSize

	go func() {
		start <- struct{}{}
		for i := 0; i < testSize; i++ {
			p, err := readerConn.ReadMessage()
			assert.NoError(t, err)
			assert.NotNil(t, p.Message)
			assert.Equal(t, uint16(64), p.Message.Id)
			assert.Equal(t, uint16(32), p.Message.Operation)
			assert.Equal(t, uint32(messageSize), p.Message.ContentLength)
			assert.Equal(t, messageSize, len(p.Content))
			assert.Equal(t, randomData, p.Content)
			packet.Put(p)
		}
		end <- struct{}{}
	}()

	<-start
	for i := 0; i < testSize; i++ {
		err := writerConn.WriteMessage(p)
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
	p.Message.Id = 64
	p.Message.Operation = 32

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	go func() {
		start <- struct{}{}
		p, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.NotNil(t, p.Message)
		assert.Equal(t, uint16(64), p.Message.Id)
		assert.Equal(t, uint16(32), p.Message.Operation)
		assert.Equal(t, uint32(0), p.Message.ContentLength)
		assert.Equal(t, 0, len(p.Content))
		end <- struct{}{}
		packet.Put(p)
	}()

	<-start
	err := writerConn.WriteMessage(p)
	assert.NoError(t, err)
	<-end

	err = readerConn.conn.Close()
	assert.NoError(t, err)

	err = writerConn.WriteMessage(p)
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
	p.Message.Id = 64
	p.Message.Operation = 32

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	go func() {
		start <- struct{}{}
		p, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.NotNil(t, p.Message)
		assert.Equal(t, uint16(64), p.Message.Id)
		assert.Equal(t, uint16(32), p.Message.Operation)
		assert.Equal(t, uint32(0), p.Message.ContentLength)
		assert.Equal(t, 0, len(p.Content))
		packet.Put(p)
		end <- struct{}{}
	}()

	<-start
	err := writerConn.WriteMessage(p)
	assert.NoError(t, err)
	<-end

	packet.Put(p)

	err = writerConn.conn.Close()
	assert.NoError(t, err)

	_, err = readerConn.ReadMessage()
	assert.ErrorIs(t, err, io.EOF)
	assert.ErrorIs(t, readerConn.Error(), io.EOF)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkSyncThroughputPipe(b *testing.B) {
	const testSize = 100000

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

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

func BenchmarkSyncThroughputNetwork(b *testing.B) {
	const testSize = 100000

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

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

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				done := make(chan struct{}, 1)
				errCh := make(chan error, 1)
				go func() {
					for i := 0; i < testSize; i++ {
						p, err := readerConn.ReadMessage()
						if err != nil {
							errCh <- err
							return
						}
						packet.Put(p)
					}
					done <- struct{}{}
				}()
				for i := 0; i < testSize; i++ {
					err = writerConn.WriteMessage(p)
					if err != nil {
						b.Fatal(err)
					}
				}
				select {
				case <-done:
					continue
				case err := <-errCh:
					b.Fatal(err)
				}
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

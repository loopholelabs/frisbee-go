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
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net"
	"testing"
)

func TestNewSync(t *testing.T) {
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: 0,
	}

	go func() {
		start <- struct{}{}
		readMessage, content, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.NotNil(t, readMessage)
		assert.Equal(t, *message, *readMessage)
		assert.Nil(t, content)
		end <- struct{}{}
	}()

	<-start
	err := writerConn.WriteMessage(message, nil)
	assert.NoError(t, err)
	<-end

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	message.ContentLength = messageSize

	go func() {
		start <- struct{}{}
		readMessage, content, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, *message, *readMessage)
		assert.Equal(t, data, *content)
		end <- struct{}{}
	}()

	<-start
	err = writerConn.WriteMessage(message, &data)
	assert.NoError(t, err)
	<-end

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestSyncLargeWrite(t *testing.T) {
	const testSize = 100000
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([][]byte, testSize)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	go func() {
		start <- struct{}{}
		for i := 0; i < testSize; i++ {
			readMessage, data, err := readerConn.ReadMessage()
			assert.NoError(t, err)
			assert.Equal(t, *message, *readMessage)
			assert.Equal(t, randomData[i], *data)
		}
		end <- struct{}{}
	}()

	<-start
	for i := 0; i < testSize; i++ {
		randomData[i] = make([]byte, messageSize)
		_, _ = rand.Read(randomData[i])
		err := writerConn.WriteMessage(message, &randomData[i])
		assert.NoError(t, err)
	}
	<-end

	err := readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestSyncRawConn(t *testing.T) {
	const testSize = 100000
	const messageSize = 32

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	l, err := net.Listen("tcp", ":3000")
	require.NoError(t, err)

	go func() {
		var err error
		reader, err = l.Accept()
		require.NoError(t, err)
		start <- struct{}{}
	}()

	writer, err = net.Dial("tcp", ":3000")
	require.NoError(t, err)
	<-start

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)
	_, _ = rand.Read(randomData)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	go func() {
		start <- struct{}{}
		for i := 0; i < testSize; i++ {
			readMessage, data, err := readerConn.ReadMessage()
			assert.NoError(t, err)
			assert.Equal(t, *message, *readMessage)
			assert.Equal(t, randomData, *data)
		}
		end <- struct{}{}
	}()

	<-start
	for i := 0; i < testSize; i++ {
		err := writerConn.WriteMessage(message, &randomData)
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

	err = l.Close()
	assert.NoError(t, err)
}

func TestSyncReadClose(t *testing.T) {
	reader, writer := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: 0,
	}

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	go func() {
		start <- struct{}{}
		readMessage, content, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.NotNil(t, readMessage)
		assert.Equal(t, *message, *readMessage)
		assert.Nil(t, content)
		end <- struct{}{}
	}()

	<-start
	err := writerConn.WriteMessage(message, nil)
	assert.NoError(t, err)
	<-end

	err = readerConn.conn.Close()
	assert.NoError(t, err)

	err = writerConn.WriteMessage(message, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, writerConn.Error(), io.ErrClosedPipe)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestSyncWriteClose(t *testing.T) {
	reader, writer := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: 0,
	}

	start := make(chan struct{}, 1)
	end := make(chan struct{}, 1)

	go func() {
		start <- struct{}{}
		readMessage, content, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.NotNil(t, readMessage)
		assert.Equal(t, *message, *readMessage)
		assert.Nil(t, content)
		end <- struct{}{}
	}()

	<-start
	err := writerConn.WriteMessage(message, nil)
	assert.NoError(t, err)
	<-end

	err = writerConn.conn.Close()
	assert.NoError(t, err)

	_, _, err = readerConn.ReadMessage()
	assert.ErrorIs(t, err, io.EOF)
	assert.ErrorIs(t, readerConn.Error(), io.EOF)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkSyncThroughputPipe32(b *testing.B) {
	const testSize = 100000
	const messageSize = 32

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.ReadMessage()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.WriteMessage(message, &randomData)
		}
		<-done
	}
	b.StopTimer()

	_ = readerConn.Close()
	_ = writerConn.Close()
}

func BenchmarkSyncThroughputPipe512(b *testing.B) {
	const testSize = 100000
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.ReadMessage()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.WriteMessage(message, &randomData)
		}
		<-done
	}
	b.StopTimer()

	_ = readerConn.Close()
	_ = writerConn.Close()
}

func BenchmarkSyncThroughputNetwork32(b *testing.B) {
	const testSize = 100000
	const messageSize = 32

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.ReadMessage()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.WriteMessage(message, &randomData)
		}
		<-done
	}
	b.StopTimer()

	_ = readerConn.Close()
	_ = writerConn.Close()
	_ = l.Close()
}

func BenchmarkSyncThroughputNetwork512(b *testing.B) {
	const testSize = 100000
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.ReadMessage()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.WriteMessage(message, &randomData)
		}
		<-done
	}
	b.StopTimer()

	_ = readerConn.Close()
	_ = writerConn.Close()
	_ = l.Close()
}

func BenchmarkSyncThroughputNetwork1024(b *testing.B) {
	const testSize = 100000
	const messageSize = 1024

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.ReadMessage()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.WriteMessage(message, &randomData)
		}
		<-done
	}
	b.StopTimer()

	_ = readerConn.Close()
	_ = writerConn.Close()
	_ = l.Close()
}

func BenchmarkSyncThroughputNetwork2048(b *testing.B) {
	const testSize = 100000
	const messageSize = 2048

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.ReadMessage()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.WriteMessage(message, &randomData)
		}
		<-done
	}
	b.StopTimer()

	_ = readerConn.Close()
	_ = writerConn.Close()
	_ = l.Close()
}

func BenchmarkSyncThroughputNetwork4096(b *testing.B) {
	const testSize = 100000
	const messageSize = 4096

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.ReadMessage()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.WriteMessage(message, &randomData)
		}
		<-done
	}
	b.StopTimer()

	_ = readerConn.Close()
	_ = writerConn.Close()
	_ = l.Close()
}

func BenchmarkSyncThroughputNetwork1mb(b *testing.B) {
	const testSize = 10
	const messageSize = 1 << 20

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.ReadMessage()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.WriteMessage(message, &randomData)
		}
		<-done
	}
	b.StopTimer()

	_ = readerConn.Close()
	_ = writerConn.Close()
	_ = l.Close()
}

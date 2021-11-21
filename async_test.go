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
	"time"
)

func TestNewAsync(t *testing.T) {
	t.Parallel()

	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: 0,
	}
	err := writerConn.WriteMessage(message, nil)
	assert.NoError(t, err)

	readMessage, content, err := readerConn.ReadMessage()
	assert.NoError(t, err)
	assert.NotNil(t, readMessage)
	assert.Equal(t, *message, *readMessage)
	assert.Nil(t, content)

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	message.ContentLength = messageSize
	err = writerConn.WriteMessage(message, &data)
	assert.NoError(t, err)

	readMessage, content, err = readerConn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, *message, *readMessage)
	assert.Equal(t, data, *content)

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

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	for i := 0; i < testSize; i++ {
		randomData[i] = make([]byte, messageSize)
		_, _ = rand.Read(randomData[i])
		err := writerConn.WriteMessage(message, &randomData[i])
		assert.NoError(t, err)
	}

	for i := 0; i < testSize; i++ {
		readMessage, data, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, *message, *readMessage)
		assert.Equal(t, randomData[i], *data)
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

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go func() {
		var err error
		reader, err = l.Accept()
		require.NoError(t, err)
		start <- struct{}{}
	}()

	writer, err = net.Dial("tcp", l.Addr().String())
	require.NoError(t, err)
	<-start

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	randomData := make([]byte, messageSize)
	_, _ = rand.Read(randomData)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: messageSize,
	}

	for i := 0; i < testSize; i++ {
		err := writerConn.WriteMessage(message, &randomData)
		assert.NoError(t, err)
	}

	for i := 0; i < testSize; i++ {
		readMessage, data, err := readerConn.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, *message, *readMessage)
		assert.Equal(t, randomData, *data)
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

	err = rawReaderConn.Close()
	assert.NoError(t, err)
	err = rawWriterConn.Close()
	assert.NoError(t, err)

	err = l.Close()
	assert.NoError(t, err)
}

func TestAsyncReadClose(t *testing.T) {
	t.Parallel()

	reader, writer := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: 0,
	}
	err := writerConn.WriteMessage(message, nil)
	assert.NoError(t, err)

	err = writerConn.Flush()
	assert.NoError(t, err)

	readMessage, content, err := readerConn.ReadMessage()
	assert.NoError(t, err)
	assert.NotNil(t, readMessage)
	assert.Equal(t, *message, *readMessage)
	assert.Nil(t, content)

	err = readerConn.conn.Close()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 100)

	err = writerConn.WriteMessage(message, nil)
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

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: 0,
	}
	err := writerConn.WriteMessage(message, nil)
	assert.NoError(t, err)

	err = writerConn.Flush()
	assert.NoError(t, err)

	readMessage, content, err := readerConn.ReadMessage()
	assert.NoError(t, err)
	assert.NotNil(t, readMessage)
	assert.Equal(t, *message, *readMessage)
	assert.Nil(t, content)

	err = writerConn.WriteMessage(message, nil)
	assert.NoError(t, err)

	err = writerConn.conn.Close()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 50)

	_, _, err = readerConn.ReadMessage()
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

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":0")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", l.Addr().String())
	<-start

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	message := &Message{
		To:            16,
		From:          32,
		Id:            64,
		Operation:     32,
		ContentLength: 0,
	}
	err := writerConn.WriteMessage(message, nil)
	assert.NoError(t, err)

	err = writerConn.Flush()
	assert.NoError(t, err)

	readMessage, content, err := readerConn.ReadMessage()
	assert.NoError(t, err)
	assert.NotNil(t, readMessage)
	assert.Equal(t, *message, *readMessage)
	assert.Nil(t, content)

	time.Sleep(defaultDeadline * 5)

	err = writerConn.Error()
	assert.NoError(t, err)

	err = writerConn.WriteMessage(message, nil)
	assert.NoError(t, err)

	err = writerConn.conn.Close()
	assert.NoError(t, err)

	time.Sleep(defaultDeadline)

	_, _, err = readerConn.ReadMessage()
	assert.ErrorIs(t, err, ConnectionClosed)
	assert.ErrorIs(t, readerConn.Error(), io.EOF)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkAsyncThroughputPipe32(b *testing.B) {
	const testSize = 100000
	const messageSize = 32

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

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

func BenchmarkAsyncThroughputPipe512(b *testing.B) {
	const testSize = 100000
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

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

func BenchmarkAsyncThroughputNetwork32(b *testing.B) {
	const testSize = 100000
	const messageSize = 32

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":0")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", l.Addr().String())
	<-start

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

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

func BenchmarkAsyncThroughputNetwork512(b *testing.B) {
	const testSize = 100000
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":0")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", l.Addr().String())
	<-start

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

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

func BenchmarkAsyncThroughputNetwork1024(b *testing.B) {
	const testSize = 100000
	const messageSize = 1024

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", l.Addr().String())
	<-start

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

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

func BenchmarkAsyncThroughputNetwork2048(b *testing.B) {
	const testSize = 100000
	const messageSize = 2048

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":0")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", l.Addr().String())
	<-start

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

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

func BenchmarkAsyncThroughputNetwork4096(b *testing.B) {
	const testSize = 100000
	const messageSize = 4096

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":0")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", l.Addr().String())
	<-start

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

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

func BenchmarkAsyncThroughputNetwork1mb(b *testing.B) {
	const testSize = 10
	const messageSize = 1 << 20

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":0")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", l.Addr().String())
	<-start

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

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

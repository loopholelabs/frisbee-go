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

func TestNewConn(t *testing.T) {
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

func TestLargeWrite(t *testing.T) {
	const testSize = 100000
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

func TestRawConn(t *testing.T) {
	const testSize = 100000
	const messageSize = 32

	emptyLogger := zerolog.New(ioutil.Discard)

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

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

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

func TestReadClose(t *testing.T) {
	reader, writer := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

	err = writerConn.WriteMessage(message, nil)
	assert.NoError(t, err)
	err = writerConn.Flush()
	assert.Error(t, err)
	assert.ErrorIs(t, writerConn.Error(), ConnectionPaused)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestWriteClose(t *testing.T) {
	reader, writer := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

	time.Sleep(time.Second)
	_, _, err = readerConn.ReadMessage()
	assert.ErrorIs(t, err, ConnectionPaused)
	assert.ErrorIs(t, readerConn.Error(), ConnectionPaused)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestBufferMessages(t *testing.T) {
	reader, writer := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

	rawWriteMessage := []byte("TEST CASE MESSAGE")

	n, err := writerConn.Write(rawWriteMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)

	err = writerConn.Flush()
	assert.NoError(t, err)

	rawReadMessage := make([]byte, len(rawWriteMessage))

	n, err = io.ReadAtLeast(readerConn, rawReadMessage, len(rawWriteMessage))
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)
	assert.Equal(t, rawWriteMessage, rawReadMessage)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestReadFrom(t *testing.T) {
	readerOne, writerOne := net.Pipe()
	readerTwo, writerTwo := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	frisbeeWriter := New(writerTwo, &emptyLogger)
	frisbeeReader := New(readerTwo, &emptyLogger)

	done := make(chan struct{}, 1)

	rawWriteMessage := []byte("TEST CASE MESSAGE")

	go func() {
		n, _ := io.Copy(frisbeeWriter, readerOne)
		assert.Equal(t, int64(len(rawWriteMessage)), n)
		done <- struct{}{}
	}()

	n, err := writerOne.Write(rawWriteMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)

	time.Sleep(time.Second)

	err = writerOne.Close()
	assert.NoError(t, err)

	err = readerOne.Close()
	assert.NoError(t, err)

	<-done

	err = frisbeeWriter.Flush()
	assert.NoError(t, err)

	rawReadMessage := make([]byte, len(rawWriteMessage))
	n, err = frisbeeReader.Read(rawReadMessage)

	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)
	assert.Equal(t, rawWriteMessage, rawReadMessage)

	err = frisbeeReader.Close()
	assert.NoError(t, err)
	err = frisbeeWriter.Close()
	assert.NoError(t, err)
}

func TestWriteTo(t *testing.T) {
	readerOne, writerOne := net.Pipe()
	readerTwo, writerTwo := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	frisbeeWriter := New(writerTwo, &emptyLogger)
	frisbeeReader := New(readerTwo, &emptyLogger)

	done := make(chan struct{}, 1)

	rawWriteMessage := []byte("TEST CASE MESSAGE")

	n, err := frisbeeWriter.Write(rawWriteMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)

	err = frisbeeWriter.Flush()
	assert.NoError(t, err)

	time.Sleep(time.Second) // Making sure that the data has propagated into the frisbee reader

	go func() {
		n, _ := io.Copy(writerOne, frisbeeReader)
		assert.Equal(t, int64(len(rawWriteMessage)), n)
		done <- struct{}{}
	}()

	rawReadMessage := make([]byte, len(rawWriteMessage))
	n, err = readerOne.Read(rawReadMessage)

	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)
	assert.Equal(t, rawWriteMessage, rawReadMessage)

	err = frisbeeWriter.Close()
	assert.NoError(t, err)

	err = frisbeeReader.Close()
	assert.NoError(t, err)

	<-done

	err = readerOne.Close()
	assert.NoError(t, err)
	err = writerOne.Close()
	assert.NoError(t, err)
}

func TestIOCopy(t *testing.T) {
	readerOne, writerOne := net.Pipe()
	readerTwo, writerTwo := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	frisbeeWriterOne := New(writerOne, &emptyLogger)
	frisbeeReaderOne := New(readerOne, &emptyLogger)

	frisbeeWriterTwo := New(writerTwo, &emptyLogger)
	frisbeeReaderTwo := New(readerTwo, &emptyLogger)

	done := make(chan struct{}, 1)

	rawWriteMessage := []byte("TEST CASE MESSAGE")

	n, err := frisbeeWriterOne.Write(rawWriteMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)

	err = frisbeeWriterOne.Flush()
	assert.NoError(t, err)

	go func() {
		n, _ := io.Copy(frisbeeWriterTwo, frisbeeReaderOne)
		assert.Equal(t, int64(len(rawWriteMessage)), n)
		done <- struct{}{}
	}()

	time.Sleep(time.Second)

	err = frisbeeWriterOne.Close()
	assert.NoError(t, err)

	err = frisbeeReaderOne.Close()
	assert.NoError(t, err)

	<-done

	err = frisbeeWriterTwo.Flush()
	assert.NoError(t, err)

	rawReadMessage := make([]byte, len(rawWriteMessage))
	n, err = frisbeeReaderTwo.Read(rawReadMessage)

	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)
	assert.Equal(t, rawWriteMessage, rawReadMessage)

	err = frisbeeReaderTwo.Close()
	assert.NoError(t, err)
	err = frisbeeWriterTwo.Close()
	assert.NoError(t, err)
}

func BenchmarkThroughputPipe32(b *testing.B) {
	const testSize = 100000
	const messageSize = 32

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

func BenchmarkThroughputPipe512(b *testing.B) {
	const testSize = 100000
	const messageSize = 512

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer := net.Pipe()

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

func BenchmarkThroughputNetwork32(b *testing.B) {
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

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

func BenchmarkThroughputNetwork512(b *testing.B) {
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

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

func BenchmarkThroughputNetwork1024(b *testing.B) {
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

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

func BenchmarkThroughputNetwork2048(b *testing.B) {
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

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

func BenchmarkThroughputNetwork4096(b *testing.B) {
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

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

func BenchmarkThroughputNetwork1mb(b *testing.B) {
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

	readerConn := New(reader, &emptyLogger)
	writerConn := New(writer, &emptyLogger)

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

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
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

func TestStreamMessages(t *testing.T) {
	t.Parallel()

	reader, writer := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	rawWriteMessage := []byte("TEST CASE MESSAGE")

	writeStream := writerConn.NewStream(1)

	n, err := writeStream.Write(rawWriteMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)

	err = writerConn.Flush()
	assert.NoError(t, err)

	rawReadMessage := make([]byte, len(rawWriteMessage))
	streamChannel := readerConn.StreamChannel()

	readStream := <-streamChannel

	n, err = io.ReadAtLeast(readStream, rawReadMessage, len(rawWriteMessage))
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)
	assert.Equal(t, rawWriteMessage, rawReadMessage)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestStreamReadFrom(t *testing.T) {
	t.Parallel()

	readerOne, writerOne := net.Pipe()
	readerTwo, writerTwo := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	frisbeeWriter := NewAsync(writerTwo, &emptyLogger)
	frisbeeReader := NewAsync(readerTwo, &emptyLogger)

	writeStream := frisbeeWriter.NewStream(1)

	done := make(chan struct{}, 1)

	rawWriteMessage := []byte("TEST CASE MESSAGE")

	go func() {
		n, _ := io.Copy(writeStream, readerOne)
		assert.Equal(t, int64(len(rawWriteMessage)), n)
		done <- struct{}{}
	}()

	n, err := writerOne.Write(rawWriteMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)

	err = writerOne.Close()
	assert.NoError(t, err)

	err = readerOne.Close()
	assert.NoError(t, err)

	<-done

	err = frisbeeWriter.Flush()
	assert.NoError(t, err)

	readStream := <-frisbeeReader.streamConnCh

	rawReadMessage := make([]byte, len(rawWriteMessage))
	n, err = readStream.Read(rawReadMessage)

	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)
	assert.Equal(t, rawWriteMessage, rawReadMessage)

	err = frisbeeReader.Close()
	assert.NoError(t, err)
	err = frisbeeWriter.Close()
	assert.NoError(t, err)
}

func TestStreamWriteTo(t *testing.T) {
	t.Parallel()

	readerOne, writerOne := net.Pipe()
	readerTwo, writerTwo := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	frisbeeWriter := NewAsync(writerTwo, &emptyLogger)
	frisbeeReader := NewAsync(readerTwo, &emptyLogger)

	streamWriter := frisbeeWriter.NewStream(1)

	done := make(chan struct{}, 1)

	rawWriteMessage := []byte("TEST CASE MESSAGE")

	n, err := streamWriter.Write(rawWriteMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)

	err = frisbeeWriter.Flush()
	assert.NoError(t, err)

	streamReader := <-frisbeeReader.streamConnCh

	go func() {
		n, _ := io.Copy(writerOne, streamReader)
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

func TestStreamIOCopy(t *testing.T) {
	var readerOne, writerOne, readerTwo, writerTwo net.Conn
	setup := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":0")

	go func() {
		readerOne, _ = l.Accept()
		setup <- struct{}{}
	}()

	writerOne, _ = net.Dial("tcp", l.Addr().String())
	<-setup

	go func() {
		readerTwo, _ = l.Accept()
		setup <- struct{}{}
	}()

	writerTwo, _ = net.Dial("tcp", l.Addr().String())
	<-setup

	emptyLogger := zerolog.New(ioutil.Discard)

	frisbeeWriterOne := NewAsync(writerOne, &emptyLogger)
	streamWriterOne := frisbeeWriterOne.NewStream(1)

	frisbeeReaderOne := NewAsync(readerOne, &emptyLogger)

	frisbeeWriterTwo := NewAsync(writerTwo, &emptyLogger)
	streamWriterTwo := frisbeeWriterTwo.NewStream(1)

	frisbeeReaderTwo := NewAsync(readerTwo, &emptyLogger)

	start := make(chan struct{}, 1)
	done := make(chan struct{}, 1)

	rawWriteMessage := []byte("TEST CASE MESSAGE")

	n, err := streamWriterOne.Write(rawWriteMessage)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)

	err = frisbeeWriterOne.Flush()
	assert.NoError(t, err)

	streamReaderOne := <-frisbeeReaderOne.streamConnCh

	go func() {
		start <- struct{}{}
		n, err := io.Copy(streamWriterTwo, streamReaderOne)
		if n != int64(len(rawWriteMessage)) {
			assert.NoError(t, err)
		}
		assert.Equal(t, int64(len(rawWriteMessage)), n)
		done <- struct{}{}
	}()

	<-start

	time.Sleep(time.Second * 3)

	err = frisbeeWriterOne.Close()
	assert.NoError(t, err)

	err = frisbeeReaderOne.Close()
	assert.NoError(t, err)

	<-done

	err = frisbeeWriterTwo.Flush()
	assert.NoError(t, err)

	streamReaderTwo := <-frisbeeReaderTwo.streamConnCh

	rawReadMessage := make([]byte, len(rawWriteMessage))
	n, err = streamReaderTwo.Read(rawReadMessage)

	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessage), n)
	assert.Equal(t, rawWriteMessage, rawReadMessage)

	err = frisbeeReaderTwo.Close()
	assert.NoError(t, err)
	err = frisbeeWriterTwo.Close()
	assert.NoError(t, err)

	err = l.Close()
	assert.NoError(t, err)
}

func TestNewStream(t *testing.T) {
	t.Parallel()

	readerOne, writerOne := net.Pipe()

	emptyLogger := zerolog.New(ioutil.Discard)

	frisbeeWriter := NewAsync(writerOne, &emptyLogger)
	frisbeeReader := NewAsync(readerOne, &emptyLogger)

	StreamWriterOne := frisbeeWriter.NewStream(1)
	StreamWriterTwo := frisbeeWriter.NewStream(2)

	rawWriteMessageOne := []byte("TEST CASE MESSAGE 1")

	n, err := StreamWriterOne.Write(rawWriteMessageOne)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessageOne), n)

	StreamReaderOne := <-frisbeeReader.streamConnCh
	rawReadMessageOne := make([]byte, len(rawWriteMessageOne))

	n, err = StreamReaderOne.Read(rawReadMessageOne)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessageOne), n)
	assert.Equal(t, rawWriteMessageOne, rawReadMessageOne)

	rawWriteMessageTwo := []byte("TEST CASE MESSAGE 2")

	n, err = StreamWriterTwo.Write(rawWriteMessageTwo)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessageTwo), n)

	StreamReaderTwo := <-frisbeeReader.streamConnCh
	rawReadMessageTwo := make([]byte, len(rawWriteMessageTwo))

	n, err = StreamReaderTwo.Read(rawReadMessageTwo)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessageTwo), n)
	assert.Equal(t, rawWriteMessageTwo, rawReadMessageTwo)

	n, err = StreamWriterOne.Write(rawWriteMessageOne)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessageOne), n)

	n, err = StreamReaderOne.Read(rawReadMessageOne)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessageOne), n)
	assert.Equal(t, rawWriteMessageOne, rawReadMessageOne)

	err = StreamWriterOne.Close()
	assert.NoError(t, err)

	n, err = StreamReaderOne.Read(rawReadMessageOne)
	assert.Error(t, err)
	assert.Equal(t, 0, n)

	n, err = StreamWriterTwo.Write(rawWriteMessageTwo)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessageTwo), n)

	n, err = StreamReaderTwo.Read(rawReadMessageTwo)
	assert.NoError(t, err)
	assert.Equal(t, len(rawWriteMessageTwo), n)
	assert.Equal(t, rawWriteMessageTwo, rawReadMessageTwo)

	err = StreamWriterOne.Async.Close()
	assert.NoError(t, err)
	err = StreamWriterTwo.Async.Close()
	assert.NoError(t, err)

	err = StreamReaderOne.Async.Close()
	assert.NoError(t, err)
	err = StreamReaderTwo.Async.Close()
	assert.NoError(t, err)
}

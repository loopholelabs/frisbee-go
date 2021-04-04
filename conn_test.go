package frisbee

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestNewConn(t *testing.T) {
	const messageSize = 512

	reader, writer := net.Pipe()

	readerConn := New(reader, nil)
	writerConn := New(writer, nil)

	readerConn.SetContext("TEST")
	assert.Equal(t, "TEST", readerConn.Context())
	assert.Nil(t, writerConn.Context())

	message := &Message{
		Id:            16,
		Operation:     32,
		Routing:       64,
		ContentLength: 0,
	}
	err := writerConn.Write(message, nil)
	assert.NoError(t, err)

	readMessage, content, err := readerConn.Read()
	assert.NoError(t, err)
	assert.Equal(t, *message, *readMessage)
	assert.Nil(t, content)

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	message.ContentLength = messageSize
	err = writerConn.Write(message, &data)
	assert.NoError(t, err)

	readMessage, content, err = readerConn.Read()
	assert.NoError(t, err)
	assert.Equal(t, *message, *readMessage)
	assert.Equal(t, data, *content)
}

func TestLargeWrite(t *testing.T) {
	const testSize = 100000
	const messageSize = 512

	reader, writer := net.Pipe()

	readerConn := New(reader, nil)
	writerConn := New(writer, nil)

	randomData := make([][]byte, testSize)

	message := &Message{
		Id:            16,
		Operation:     32,
		Routing:       64,
		ContentLength: messageSize,
	}

	for i := 0; i < testSize; i++ {
		randomData[i] = make([]byte, messageSize)
		_, _ = rand.Read(randomData[i])
		err := writerConn.Write(message, &randomData[i])
		assert.NoError(t, err)
	}

	for i := 0; i < testSize; i++ {
		readMessage, data, err := readerConn.Read()
		assert.NoError(t, err)
		assert.Equal(t, *message, *readMessage)
		assert.Equal(t, randomData[i], *data)
	}

	err := readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func BenchmarkThroughputPipe32(b *testing.B) {
	const testSize = 100000
	const messageSize = 32

	reader, writer := net.Pipe()

	readerConn := New(reader, nil)
	writerConn := New(writer, nil)

	randomData := make([]byte, messageSize)

	message := &Message{
		Id:            16,
		Operation:     32,
		Routing:       64,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.Read()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.Write(message, &randomData)
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

	reader, writer := net.Pipe()

	readerConn := New(reader, nil)
	writerConn := New(writer, nil)

	randomData := make([]byte, messageSize)

	message := &Message{
		Id:            16,
		Operation:     32,
		Routing:       64,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.Read()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.Write(message, &randomData)
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

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := New(reader, nil)
	writerConn := New(writer, nil)

	randomData := make([]byte, messageSize)

	message := &Message{
		Id:            16,
		Operation:     32,
		Routing:       64,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.Read()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.Write(message, &randomData)
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

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := New(reader, nil)
	writerConn := New(writer, nil)

	randomData := make([]byte, messageSize)

	message := &Message{
		Id:            16,
		Operation:     32,
		Routing:       64,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.Read()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.Write(message, &randomData)
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

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := New(reader, nil)
	writerConn := New(writer, nil)

	randomData := make([]byte, messageSize)

	message := &Message{
		Id:            16,
		Operation:     32,
		Routing:       64,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.Read()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.Write(message, &randomData)
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

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := New(reader, nil)
	writerConn := New(writer, nil)

	randomData := make([]byte, messageSize)

	message := &Message{
		Id:            16,
		Operation:     32,
		Routing:       64,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.Read()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.Write(message, &randomData)
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

	var reader, writer net.Conn
	start := make(chan struct{}, 1)

	l, _ := net.Listen("tcp", ":3000")

	go func() {
		reader, _ = l.Accept()
		start <- struct{}{}
	}()

	writer, _ = net.Dial("tcp", ":3000")
	<-start

	readerConn := New(reader, nil)
	writerConn := New(writer, nil)

	randomData := make([]byte, messageSize)

	message := &Message{
		Id:            16,
		Operation:     32,
		Routing:       64,
		ContentLength: messageSize,
	}

	done := make(chan struct{}, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for i := 0; i < testSize; i++ {
				readMessage, data, _ := readerConn.Read()
				_ = data
				_ = readMessage
			}
			done <- struct{}{}
		}()
		for i := 0; i < testSize; i++ {
			_ = writerConn.Write(message, &randomData)
		}
		<-done
	}
	b.StopTimer()

	_ = readerConn.Close()
	_ = writerConn.Close()
	_ = l.Close()
}

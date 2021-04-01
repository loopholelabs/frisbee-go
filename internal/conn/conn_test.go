package conn

import (
	"crypto/rand"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/stretchr/testify/assert"
	"hash/crc32"
	"net"
	"testing"
)

func TestNewConn(t *testing.T) {
	const messageSize = 512

	reader, writer := net.Pipe()

	readerConn := New(reader)
	writerConn := New(writer)

	readerConn.SetContext("TEST")
	assert.Equal(t, "TEST", readerConn.Context())
	assert.Nil(t, writerConn.Context())

	message := &protocol.MessageV0{
		Id:            16,
		Operation:     32,
		Routing:       64,
		ContentLength: 0,
	}
	err := writerConn.Write(message, nil)
	assert.NoError(t, err)

	readMessage, content, err := readerConn.Read()
	assert.NoError(t, err)
	assert.Equal(t, *message, readMessage)
	assert.Nil(t, content)

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	message.ContentLength = messageSize
	err = writerConn.Write(message, &data)

	readMessage, content, err = readerConn.Read()
	assert.NoError(t, err)
	assert.Equal(t, *message, readMessage)
	assert.Equal(t, data, content)
}

func TestLargeWrite(t *testing.T) {
	const testSize = 1 << 7

	var reader, writer net.Conn
	accepted := make(chan struct{}, 1)

	l, err := net.Listen("tcp", ":3000")
	assert.NoError(t, err)

	go func() {
		for {
			reader, err = l.Accept()
			assert.NoError(t, err)
			accepted <- struct{}{}
		}
	}()

	writer, err = net.Dial("tcp", ":3000")
	assert.NoError(t, err)

	<-accepted

	readerConn := New(reader)
	writerConn := New(writer)

	done := make(chan struct{}, 1)

	randomData := make([][]byte, testSize)

	go func() {
		for i := 0; i < testSize; i++ {
			readMessage, data, err := readerConn.Read()
			assert.NoError(t, err)
			assert.Equal(t, uint32(i+1), readMessage.Id)
			assert.Equal(t, crc32.ChecksumIEEE(randomData[i]), crc32.ChecksumIEEE(data), "Message ID: %d incorrect", readMessage.Id)
		}
		done <- struct{}{}
	}()

	for i := 0; i < testSize; i++ {
		randomData[i] = make([]byte, i+1)
		_, _ = rand.Read(randomData[i])
		message := &protocol.MessageV0{
			Id:            uint32(i + 1),
			Operation:     32,
			Routing:       64,
			ContentLength: uint32(i + 1),
		}
		err = writerConn.Write(message, &randomData[i])
		assert.NoError(t, err)
	}
	<-done
}

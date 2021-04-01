package conn

import (
	"crypto/rand"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, *message, *readMessage)
	assert.Nil(t, content)

	data := make([]byte, messageSize)
	_, _ = rand.Read(data)

	message.ContentLength = messageSize
	err = writerConn.Write(message, &data)

	readMessage, content, err = readerConn.Read()
	assert.NoError(t, err)
	assert.Equal(t, *message, *readMessage)
	assert.Equal(t, data, *content)
}

func TestLargeWrite(t *testing.T) {
	const testSize = defaultSize / 2
	const messageSize = defaultSize / 2

	reader, writer := net.Pipe()

	readerConn := New(reader)
	writerConn := New(writer)

	randomData := make([][]byte, testSize)

	message := &protocol.MessageV0{
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
}

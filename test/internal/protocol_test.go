package test

import (
	"crypto/rand"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	assert.Equal(t, protocol.DefaultProtocolVersion, protocol.DefaultOptions().Version)
}

func TestNewMessageHandler(t *testing.T) {
	_, err := protocol.NewMessageHandler(&protocol.MessageOptions{
		Version: 0x0000,
	})
	assert.NotEqual(t, nil, err)
}

func TestEncodeDecodeV0(t *testing.T) {
	defaultOptions := protocol.DefaultOptions()

	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0, err := protocol.NewMessageHandler(defaultOptions)
	assert.Equal(t, nil, err)
	assert.Equal(t, defaultOptions, handlerV0.Options)

	encodedBytes, err := handlerV0.Encode(protocol.MessagePing, randomData)
	assert.Equal(t, nil, err)

	message, err := handlerV0.Decode(encodedBytes)
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(512), message.Length())
	assert.Equal(t, protocol.DefaultProtocolVersion, message.Ver())
	assert.Equal(t, protocol.MessagePing, message.Type())

	messageData, err := message.Data()
	assert.Equal(t, err, nil)
	assert.Equal(t, messageData, randomData)
}

func TestUnsafeEncodeDecodeV0(t *testing.T) {
	defaultOptions := protocol.DefaultOptions()
	defaultOptions.Unsafe = true

	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0, err := protocol.NewMessageHandler(defaultOptions)
	assert.Equal(t, nil, err)
	assert.Equal(t, defaultOptions, handlerV0.Options)

	encodedBytes, err := handlerV0.Encode(protocol.MessagePing, randomData)
	assert.Equal(t, nil, err)

	message, err := handlerV0.Decode(encodedBytes)
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(512), message.Length())
	assert.Equal(t, protocol.DefaultProtocolVersion, message.Ver())
	assert.Equal(t, protocol.MessagePing, message.Type())

	messageData, err := message.Data()
	assert.Equal(t, err, nil)
	assert.Equal(t, messageData, randomData)
}

func BenchmarkEncode(b *testing.B) {
	defaultOptions := protocol.DefaultOptions()

	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0, _ := protocol.NewMessageHandler(defaultOptions)

	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Encode(protocol.MessagePing, randomData)
	}
}

func BenchmarkDecode(b *testing.B) {
	defaultOptions := protocol.DefaultOptions()

	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0, _ := protocol.NewMessageHandler(defaultOptions)
	encodedMessage, _ := handlerV0.Encode(protocol.MessagePing, randomData)

	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Decode(encodedMessage)
	}
}

func BenchmarkUnsafeEncode(b *testing.B) {
	defaultOptions := protocol.DefaultOptions()
	defaultOptions.Unsafe = true

	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0, _ := protocol.NewMessageHandler(defaultOptions)

	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Encode(protocol.MessagePing, randomData)
	}
}

func BenchmarkUnsafeDecode(b *testing.B) {
	defaultOptions := protocol.DefaultOptions()
	defaultOptions.Unsafe = true

	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0, _ := protocol.NewMessageHandler(defaultOptions)
	encodedMessage, _ := handlerV0.Encode(protocol.MessagePing, randomData)

	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Decode(encodedMessage)
	}
}

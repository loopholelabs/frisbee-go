package test

import (
	"crypto/rand"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultHandler(t *testing.T) {
	defaultHandler := protocol.NewDefaultHandler()
	assert.Equal(t, false, defaultHandler.Unsafe)
	assert.Equal(t, defaultHandler, protocol.NewV0Handler(false))
}

func TestEncodeDecodeV0(t *testing.T) {
	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0 := protocol.NewV0Handler(false)
	assert.Equal(t, false, handlerV0.Unsafe)

	encodedBytes, err := handlerV0.Encode(protocol.MessagePacket, 0, randomData)
	assert.Equal(t, nil, err)

	message, err := handlerV0.Decode(encodedBytes)
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(512), message.ContentLength)
	assert.Equal(t, protocol.Version0, message.Version)
	assert.Equal(t, uint32(0), message.Routing)
	assert.Equal(t, protocol.MessagePacket, message.Operation)
	assert.Equal(t, randomData, message.Content)
}

func TestUnsafeEncodeDecodeV0(t *testing.T) {

	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0 := protocol.NewV0Handler(true)
	assert.Equal(t, true, handlerV0.Unsafe)

	encodedBytes, err := handlerV0.Encode(protocol.MessagePacket, 0, randomData)
	assert.Equal(t, nil, err)

	message, err := handlerV0.Decode(encodedBytes)
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(512), message.ContentLength)
	assert.Equal(t, protocol.Version0, message.Version)
	assert.Equal(t, uint32(0), message.Routing)
	assert.Equal(t, protocol.MessagePacket, message.Operation)
	assert.Equal(t, randomData, message.Content)

	emptyData := make([]byte, 0)
	emptyEncodedBytes, err := handlerV0.Encode(protocol.MessagePing, 0, emptyData)
	assert.Equal(t, nil, err)

	emptyMessage, err := handlerV0.Decode(emptyEncodedBytes)
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(0), emptyMessage.ContentLength)
	assert.Equal(t, protocol.Version0, emptyMessage.Version)
	assert.Equal(t, uint32(0), emptyMessage.Routing)
	assert.Equal(t, protocol.MessagePing, emptyMessage.Operation)
	assert.Equal(t, emptyData, emptyMessage.Content)
}

func BenchmarkEncode(b *testing.B) {
	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0 := protocol.NewV0Handler(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Encode(protocol.MessagePacket, 0, randomData)
	}
}

func BenchmarkDecode(b *testing.B) {
	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0 := protocol.NewV0Handler(false)
	encodedMessage, _ := handlerV0.Encode(protocol.MessagePacket, 0, randomData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Decode(encodedMessage)
	}
}

func BenchmarkUnsafeEncode(b *testing.B) {
	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0 := protocol.NewV0Handler(true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Encode(protocol.MessagePacket, 0, randomData)
	}
}

func BenchmarkUnsafeDecode(b *testing.B) {
	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0 := protocol.NewV0Handler(true)
	encodedMessage, _ := handlerV0.Encode(protocol.MessagePacket, 0, randomData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Decode(encodedMessage)
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0 := protocol.NewV0Handler(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := handlerV0.Encode(protocol.MessagePacket, 0, randomData)
		_, _ = handlerV0.Decode(encodedMessage)
	}
}

func BenchmarkUnsafeEncodeDecode(b *testing.B) {
	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	handlerV0 := protocol.NewV0Handler(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := handlerV0.Encode(protocol.MessagePacket, 0, randomData)
		_, _ = handlerV0.Decode(encodedMessage)
	}
}

package test

import (
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultHandler(t *testing.T) {
	defaultHandler := protocol.NewDefaultHandler()
	assert.Equal(t, defaultHandler, protocol.NewV0Handler())
}

func TestEncodeDecodeHandlerV0(t *testing.T) {
	handlerV0 := protocol.NewV0Handler()

	encodedBytes, err := handlerV0.Encode(16, protocol.MessagePacket, 0, 512)
	assert.Equal(t, nil, err)

	message, err := handlerV0.Decode(encodedBytes[:])
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(512), message.ContentLength)
	assert.Equal(t, protocol.Version0, message.Version)
	assert.Equal(t, uint16(16), message.Id)
	assert.Equal(t, uint32(0), message.Routing)
	assert.Equal(t, protocol.MessagePacket, message.Operation)

	emptyEncodedBytes, err := handlerV0.Encode(16, protocol.MessagePing, 0, 0)
	assert.Equal(t, nil, err)

	emptyMessage, err := handlerV0.Decode(emptyEncodedBytes[:])
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(0), emptyMessage.ContentLength)
	assert.Equal(t, protocol.Version0, emptyMessage.Version)
	assert.Equal(t, uint16(16), message.Id)
	assert.Equal(t, uint32(0), emptyMessage.Routing)
	assert.Equal(t, protocol.MessagePing, emptyMessage.Operation)

	invalidMessage, err := handlerV0.Decode(emptyEncodedBytes[8:])
	assert.NotEqual(t, nil, err)
	assert.Equal(t, errors.New("invalid buffer length").Error(), err.Error())
	assert.Equal(t, uint32(0), invalidMessage.ContentLength)
	assert.Equal(t, uint8(0), invalidMessage.Version)
	assert.Equal(t, uint16(0), invalidMessage.Id)
	assert.Equal(t, uint32(0), invalidMessage.Routing)
	assert.Equal(t, uint16(0), invalidMessage.Operation)

}

func TestEncodeDecodeV0(t *testing.T) {
	encodedBytes, err := protocol.EncodeV0(16, protocol.MessagePacket, 0, 512)
	assert.Equal(t, nil, err)

	message, err := protocol.DecodeV0(encodedBytes[:])
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(512), message.ContentLength)
	assert.Equal(t, protocol.Version0, message.Version)
	assert.Equal(t, uint16(16), message.Id)
	assert.Equal(t, uint32(0), message.Routing)
	assert.Equal(t, protocol.MessagePacket, message.Operation)

	emptyEncodedBytes, err := protocol.EncodeV0(16, protocol.MessagePing, 0, 0)
	assert.Equal(t, nil, err)

	emptyMessage, err := protocol.DecodeV0(emptyEncodedBytes[:])
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(0), emptyMessage.ContentLength)
	assert.Equal(t, protocol.Version0, emptyMessage.Version)
	assert.Equal(t, uint16(16), emptyMessage.Id)
	assert.Equal(t, uint32(0), emptyMessage.Routing)
	assert.Equal(t, protocol.MessagePing, emptyMessage.Operation)

	invalidMessage, err := protocol.DecodeV0(emptyEncodedBytes[1:])
	assert.NotEqual(t, nil, err)
	assert.Equal(t, errors.New("invalid buffer length").Error(), err.Error())
	assert.Equal(t, uint32(0), invalidMessage.ContentLength)
	assert.Equal(t, uint8(0), invalidMessage.Version)
	assert.Equal(t, uint16(0), invalidMessage.Id)
	assert.Equal(t, uint32(0), invalidMessage.Routing)
	assert.Equal(t, uint16(0), invalidMessage.Operation)
}

func BenchmarkEncodeHandler(b *testing.B) {

	handlerV0 := protocol.NewV0Handler()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Encode(uint16(i), protocol.MessagePacket, 0, 512)
	}
}

func BenchmarkDecodeHandler(b *testing.B) {
	handlerV0 := protocol.NewV0Handler()
	encodedMessage, _ := handlerV0.Encode(0, protocol.MessagePacket, 0, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Decode(encodedMessage[:])
	}
}

func BenchmarkEncodeDecodeHandler(b *testing.B) {
	handlerV0 := protocol.NewV0Handler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := handlerV0.Encode(uint16(i), protocol.MessagePacket, 0, 512)
		_, _ = handlerV0.Decode(encodedMessage[:])
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = protocol.EncodeV0(uint16(i), protocol.MessagePacket, 0, 512)
	}
}

func BenchmarkDecode(b *testing.B) {
	encodedMessage, _ := protocol.EncodeV0(0, protocol.MessagePacket, 0, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = protocol.DecodeV0(encodedMessage[:])
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := protocol.EncodeV0(uint16(i), protocol.MessagePacket, 0, 512)
		_, _ = protocol.DecodeV0(encodedMessage[:])
	}
}

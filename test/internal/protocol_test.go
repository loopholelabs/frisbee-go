package test

import (
	"bytes"
	"crypto/rand"
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

	encodedBytes, err := handlerV0.Encode(protocol.MessagePacket, 0, 512)
	assert.Equal(t, nil, err)

	message, err := handlerV0.Decode(encodedBytes[:])
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(512), message.ContentLength)
	assert.Equal(t, protocol.Version0, message.Version)
	assert.Equal(t, uint32(0), message.Routing)
	assert.Equal(t, protocol.MessagePacket, message.Operation)

	emptyEncodedBytes, err := handlerV0.Encode(protocol.MessagePing, 0, 0)
	assert.Equal(t, nil, err)

	emptyMessage, err := handlerV0.Decode(emptyEncodedBytes[:])
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(0), emptyMessage.ContentLength)
	assert.Equal(t, protocol.Version0, emptyMessage.Version)
	assert.Equal(t, uint32(0), emptyMessage.Routing)
	assert.Equal(t, protocol.MessagePing, emptyMessage.Operation)

	invalidMessage, err := handlerV0.Decode(emptyEncodedBytes[1:])
	assert.NotEqual(t, nil, err)
	assert.Equal(t, errors.New("Invalid Buffer Length").Error(), err.Error())
	assert.Equal(t, uint32(0), invalidMessage.ContentLength)
	assert.Equal(t, uint8(0), invalidMessage.Version)
	assert.Equal(t, uint32(0), invalidMessage.Routing)
	assert.Equal(t, uint16(0), invalidMessage.Operation)

}

func TestEncodeDecodeV0(t *testing.T) {
	encodedBytes, err := protocol.EncodeV0(protocol.MessagePacket, 0, 512)
	assert.Equal(t, nil, err)

	message, err := protocol.DecodeV0(encodedBytes[:])
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(512), message.ContentLength)
	assert.Equal(t, protocol.Version0, message.Version)
	assert.Equal(t, uint32(0), message.Routing)
	assert.Equal(t, protocol.MessagePacket, message.Operation)

	emptyEncodedBytes, err := protocol.EncodeV0(protocol.MessagePing, 0, 0)
	assert.Equal(t, nil, err)

	emptyMessage, err := protocol.DecodeV0(emptyEncodedBytes[:])
	assert.Equal(t, nil, err)
	assert.Equal(t, uint32(0), emptyMessage.ContentLength)
	assert.Equal(t, protocol.Version0, emptyMessage.Version)
	assert.Equal(t, uint32(0), emptyMessage.Routing)
	assert.Equal(t, protocol.MessagePing, emptyMessage.Operation)

	invalidMessage, err := protocol.DecodeV0(emptyEncodedBytes[1:])
	assert.NotEqual(t, nil, err)
	assert.Equal(t, errors.New("Invalid Buffer Length").Error(), err.Error())
	assert.Equal(t, uint32(0), invalidMessage.ContentLength)
	assert.Equal(t, uint8(0), invalidMessage.Version)
	assert.Equal(t, uint32(0), invalidMessage.Routing)
	assert.Equal(t, uint16(0), invalidMessage.Operation)
}

func TestWriteHandler(t *testing.T) {
	handlerV0 := protocol.NewV0Handler()

	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	encodedBytes, err := handlerV0.Encode(protocol.MessagePacket, 0, 512)
	assert.Equal(t, nil, err)

	correctData := append(encodedBytes[:], randomData...)

	var buf bytes.Buffer

	n, err := handlerV0.Write(encodedBytes, &randomData, &buf)
	assert.Equal(t, nil, err)
	assert.Equal(t, protocol.HeaderLengthV0+512, n)

	assert.Equal(t, correctData, buf.Bytes())
}

func TestWrite(t *testing.T) {
	randomData := make([]byte, 512)
	_, _ = rand.Read(randomData)

	encodedBytes, err := protocol.EncodeV0(protocol.MessagePacket, 0, 512)
	assert.Equal(t, nil, err)

	correctData := append(encodedBytes[:], randomData...)

	var buf bytes.Buffer

	n, err := protocol.Write(encodedBytes, &randomData, &buf)
	assert.Equal(t, nil, err)
	assert.Equal(t, protocol.HeaderLengthV0+512, n)

	assert.Equal(t, correctData, buf.Bytes())
}

func BenchmarkEncodeHandler(b *testing.B) {

	handlerV0 := protocol.NewV0Handler()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Encode(protocol.MessagePacket, 0, 512)
	}
}

func BenchmarkDecodeHandler(b *testing.B) {
	handlerV0 := protocol.NewV0Handler()
	encodedMessage, _ := handlerV0.Encode(protocol.MessagePacket, 0, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Decode(encodedMessage[:])
	}
}

func BenchmarkEncodeDecodeHandler(b *testing.B) {
	handlerV0 := protocol.NewV0Handler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := handlerV0.Encode(protocol.MessagePacket, 0, 512)
		_, _ = handlerV0.Decode(encodedMessage[:])
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = protocol.EncodeV0(protocol.MessagePacket, 0, 512)
	}
}

func BenchmarkDecode(b *testing.B) {
	encodedMessage, _ := protocol.EncodeV0(protocol.MessagePacket, 0, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = protocol.DecodeV0(encodedMessage[:])
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := protocol.EncodeV0(protocol.MessagePacket, 0, 512)
		_, _ = protocol.DecodeV0(encodedMessage[:])
	}
}

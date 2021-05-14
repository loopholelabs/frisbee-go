package protocol

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidVersion(t *testing.T) {
	assert.Equal(t, true, validVersion(Version0))
	assert.NotEqual(t, true, validVersion(uint16(0x00)))
}

func TestMessageV0EncodeDecode(t *testing.T) {
	message := &MessageV0{
		From:          uint32(16),
		To:            uint32(32),
		Id:            uint32(64),
		Operation:     MessagePacket,
		ContentLength: uint64(0),
	}

	correct := [HeaderLengthV0]byte{}

	binary.BigEndian.PutUint16(correct[6:8], Version0)
	binary.BigEndian.PutUint32(correct[8:12], uint32(16))
	binary.BigEndian.PutUint32(correct[12:16], uint32(32))
	binary.BigEndian.PutUint32(correct[16:20], uint32(64))
	binary.BigEndian.PutUint32(correct[20:24], MessagePacket)
	binary.BigEndian.PutUint64(correct[24:32], uint64(0))

	encoded, err := message.Encode()
	assert.NoError(t, err)
	assert.Equal(t, correct, encoded)

	decoderMessage := &MessageV0{}

	err = decoderMessage.Decode(encoded)
	assert.NoError(t, err)
	assert.Equal(t, message, decoderMessage)

}

func TestDefaultHandler(t *testing.T) {
	defaultHandler := NewDefaultHandler()
	assert.Equal(t, defaultHandler, NewV0Handler())
}

func TestEncodeDecodeHandlerV0(t *testing.T) {
	handlerV0 := NewV0Handler()

	encodedBytes, err := handlerV0.Encode(16, 32, 64, MessagePacket, 512)
	assert.NoError(t, err)

	message, err := handlerV0.Decode(encodedBytes[:])
	assert.NoError(t, err)
	assert.Equal(t, uint64(512), message.ContentLength)
	assert.Equal(t, uint32(64), message.Id)
	assert.Equal(t, uint32(32), message.To)
	assert.Equal(t, uint32(16), message.From)
	assert.Equal(t, MessagePacket, message.Operation)

	emptyEncodedBytes, err := handlerV0.Encode(16, 32, 64, MessagePing, 0)
	assert.Equal(t, nil, err)

	emptyMessage, err := handlerV0.Decode(emptyEncodedBytes[:])
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), emptyMessage.ContentLength)
	assert.Equal(t, uint32(64), emptyMessage.Id)
	assert.Equal(t, uint32(32), emptyMessage.To)
	assert.Equal(t, uint32(16), emptyMessage.From)
	assert.Equal(t, MessagePing, emptyMessage.Operation)

	invalidMessage, err := handlerV0.Decode(emptyEncodedBytes[8:])
	assert.Error(t, err)
	assert.Equal(t, errors.New("invalid buffer length").Error(), err.Error())
	assert.Equal(t, uint64(0), invalidMessage.ContentLength)
	assert.Equal(t, uint32(0), invalidMessage.Id)
	assert.Equal(t, uint32(0), invalidMessage.To)
	assert.Equal(t, uint32(0), invalidMessage.From)
}

func TestEncodeDecodeV0(t *testing.T) {
	encodedBytes, err := EncodeV0(16, 32, 64, MessagePong, 512)
	assert.Equal(t, nil, err)

	message, err := DecodeV0(encodedBytes[:])
	assert.NoError(t, err)
	assert.Equal(t, uint64(512), message.ContentLength)
	assert.Equal(t, uint32(64), message.Id)
	assert.Equal(t, uint32(32), message.To)
	assert.Equal(t, uint32(16), message.From)
	assert.Equal(t, MessagePong, message.Operation)

	emptyEncodedBytes, err := EncodeV0(16, 32, 64, MessagePing, 0)
	assert.Equal(t, nil, err)

	emptyMessage, err := DecodeV0(emptyEncodedBytes[:])
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), emptyMessage.ContentLength)
	assert.Equal(t, uint32(64), emptyMessage.Id)
	assert.Equal(t, uint32(32), emptyMessage.To)
	assert.Equal(t, uint32(16), emptyMessage.From)
	assert.Equal(t, MessagePing, emptyMessage.Operation)

	invalidMessage, err := DecodeV0(emptyEncodedBytes[1:])
	assert.Error(t, err)
	assert.Equal(t, errors.New("invalid buffer length").Error(), err.Error())
	assert.Equal(t, uint64(0), invalidMessage.ContentLength)
	assert.Equal(t, uint32(0), invalidMessage.Id)
	assert.Equal(t, uint32(0), invalidMessage.To)
	assert.Equal(t, uint32(0), invalidMessage.From)
}

func BenchmarkEncodeHandler(b *testing.B) {

	handlerV0 := NewV0Handler()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Encode(uint32(i), uint32(i), uint32(i), MessagePacket, 512)
	}
}

func BenchmarkDecodeHandler(b *testing.B) {
	handlerV0 := NewV0Handler()
	encodedMessage, _ := handlerV0.Encode(0, 0, 0, MessagePacket, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handlerV0.Decode(encodedMessage[:])
	}
}

func BenchmarkEncodeDecodeHandler(b *testing.B) {
	handlerV0 := NewV0Handler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := handlerV0.Encode(uint32(i), uint32(i), uint32(i), MessagePacket, 512)
		_, _ = handlerV0.Decode(encodedMessage[:])
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = EncodeV0(uint32(i), uint32(i), uint32(i), MessagePacket, 512)
	}
}

func BenchmarkDecode(b *testing.B) {
	encodedMessage, _ := EncodeV0(0, 0, 0, MessagePacket, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeV0(encodedMessage[:])
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := EncodeV0(uint32(i), uint32(i), uint32(i), MessagePacket, 512)
		_, _ = DecodeV0(encodedMessage[:])
	}
}

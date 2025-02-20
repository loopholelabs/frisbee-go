// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageEncodeDecode(t *testing.T) {
	t.Parallel()

	message := &Metadata{
		Magic:         PacketMagicHeader,
		Id:            uint16(64),
		Operation:     PacketProbe,
		ContentLength: uint32(0),
	}

	correct := NewBuffer()

	binary.BigEndian.PutUint16(correct[MagicOffset:MagicOffset+MagicSize], PacketMagicHeader)
	binary.BigEndian.PutUint16(correct[IdOffset:IdOffset+IdSize], uint16(64))
	binary.BigEndian.PutUint16(correct[OperationOffset:OperationOffset+OperationSize], PacketProbe)
	binary.BigEndian.PutUint32(correct[ContentLengthOffset:ContentLengthOffset+ContentLengthSize], uint32(0))

	encoded, err := message.Encode()
	require.NoError(t, err)
	assert.Equal(t, correct, encoded)

	decoderMessage := &Metadata{}

	err = decoderMessage.Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, message, decoderMessage)
}

func TestEncodeDecode(t *testing.T) {
	t.Parallel()

	encodedBytes, err := Encode(64, PacketPong, 512)
	assert.Equal(t, nil, err)

	message, err := Decode(encodedBytes[:])
	require.NoError(t, err)
	assert.Equal(t, PacketMagicHeader, message.Magic)
	assert.Equal(t, uint32(512), message.ContentLength)
	assert.Equal(t, uint16(64), message.Id)
	assert.Equal(t, PacketPong, message.Operation)

	emptyEncodedBytes, err := Encode(64, PacketPing, 0)
	assert.Equal(t, nil, err)

	emptyMessage, err := Decode(emptyEncodedBytes[:])
	require.NoError(t, err)
	assert.Equal(t, PacketMagicHeader, message.Magic)
	assert.Equal(t, uint32(0), emptyMessage.ContentLength)
	assert.Equal(t, uint16(64), emptyMessage.Id)
	assert.Equal(t, PacketPing, emptyMessage.Operation)

	invalidMessage, err := Decode(emptyEncodedBytes[1:])
	require.Error(t, err)
	assert.ErrorIs(t, InvalidBufferLengthErr, err)
	assert.Nil(t, invalidMessage)
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Encode(uint16(i), PacketProbe, 512)
	}
}

func BenchmarkDecode(b *testing.B) {
	encodedMessage, _ := Encode(0, PacketProbe, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decode(encodedMessage[:])
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := Encode(uint16(i), PacketProbe, 512)
		_, _ = Decode(encodedMessage[:])
	}
}

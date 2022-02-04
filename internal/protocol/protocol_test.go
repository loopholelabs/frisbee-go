/*
	Copyright 2022 Loophole Labs

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

package protocol

import (
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMessageEncodeDecode(t *testing.T) {
	t.Parallel()

	message := &Message{
		Id:            uint16(64),
		Operation:     MessagePacket,
		ContentLength: uint32(0),
	}

	correct := [MessageSize]byte{}

	binary.BigEndian.PutUint16(correct[IdOffset:IdOffset+IdSize], uint16(64))
	binary.BigEndian.PutUint16(correct[OperationOffset:OperationOffset+OperationSize], MessagePacket)
	binary.BigEndian.PutUint32(correct[ContentLengthOffset:ContentLengthOffset+ContentLengthSize], uint32(0))

	encoded, err := message.Encode()
	require.NoError(t, err)
	assert.Equal(t, correct, encoded)

	decoderMessage := &Message{}

	err = decoderMessage.Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, message, decoderMessage)
}

func TestDefaultHandler(t *testing.T) {
	t.Parallel()

	defaultHandler := NewDefaultHandler()
	assert.Equal(t, defaultHandler, NewHandler())
}

func TestEncodeDecodeHandler(t *testing.T) {
	t.Parallel()

	handler := NewHandler()

	encodedBytes, err := handler.Encode(64, MessagePacket, 512)
	assert.NoError(t, err)

	message, err := handler.Decode(encodedBytes[:])
	require.NoError(t, err)
	assert.Equal(t, uint32(512), message.ContentLength)
	assert.Equal(t, uint16(64), message.Id)
	assert.Equal(t, MessagePacket, message.Operation)

	emptyEncodedBytes, err := handler.Encode(64, MessagePing, 0)
	assert.Equal(t, nil, err)

	emptyMessage, err := handler.Decode(emptyEncodedBytes[:])
	require.NoError(t, err)
	assert.Equal(t, uint32(0), emptyMessage.ContentLength)
	assert.Equal(t, uint16(64), emptyMessage.Id)
	assert.Equal(t, MessagePing, emptyMessage.Operation)

	invalidMessage, err := handler.Decode(emptyEncodedBytes[8:])
	require.Error(t, err)
	assert.ErrorIs(t, InvalidBufferLength, err)
	assert.Equal(t, uint32(0), invalidMessage.ContentLength)
	assert.Equal(t, uint16(0), invalidMessage.Id)
}

func TestEncodeDecode(t *testing.T) {
	t.Parallel()

	encodedBytes, err := Encode(64, MessagePong, 512)
	assert.Equal(t, nil, err)

	message, err := Decode(encodedBytes[:])
	require.NoError(t, err)
	assert.Equal(t, uint32(512), message.ContentLength)
	assert.Equal(t, uint16(64), message.Id)
	assert.Equal(t, MessagePong, message.Operation)

	emptyEncodedBytes, err := Encode(64, MessagePing, 0)
	assert.Equal(t, nil, err)

	emptyMessage, err := Decode(emptyEncodedBytes[:])
	require.NoError(t, err)
	assert.Equal(t, uint32(0), emptyMessage.ContentLength)
	assert.Equal(t, uint16(64), emptyMessage.Id)
	assert.Equal(t, MessagePing, emptyMessage.Operation)

	invalidMessage, err := Decode(emptyEncodedBytes[1:])
	require.Error(t, err)
	assert.ErrorIs(t, InvalidBufferLength, err)
	assert.Equal(t, uint32(0), invalidMessage.ContentLength)
	assert.Equal(t, uint16(0), invalidMessage.Id)
}

func BenchmarkEncodeHandler(b *testing.B) {
	handler := NewHandler()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.Encode(uint16(i), MessagePacket, 512)
	}
}

func BenchmarkDecodeHandler(b *testing.B) {
	handler := NewHandler()
	encodedMessage, _ := handler.Encode(0, MessagePacket, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.Decode(encodedMessage[:])
	}
}

func BenchmarkEncodeDecodeHandler(b *testing.B) {
	handler := NewHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := handler.Encode(uint16(i), MessagePacket, 512)
		_, _ = handler.Decode(encodedMessage[:])
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Encode(uint16(i), MessagePacket, 512)
	}
}

func BenchmarkDecode(b *testing.B) {
	encodedMessage, _ := Encode(0, MessagePacket, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decode(encodedMessage[:])
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodedMessage, _ := Encode(uint16(i), MessagePacket, 512)
		_, _ = Decode(encodedMessage[:])
	}
}

/*
	Copyright 2021 Loophole Labs

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
	"bytes"
	"encoding/binary"
	"github.com/loophole-labs/frisbee/internal/errors"
	"unsafe"
)

const (
	ENCODE errors.ErrorContext = "error while encoding message"
	DECODE errors.ErrorContext = "error while decoding message"
)

var (
	InvalidMessageVersion = errors.New("invalid message version")
	InvalidBufferLength   = errors.New("invalid buffer length")
)

var (
	ReservedBytes = []byte{byte(0xF >> 24), byte(0xB >> 16), byte(0xE >> 8), byte(0xE)}
)

const (
	MessagePing   = uint32(10) // PING
	MessagePong   = uint32(11) // PONG
	MessagePacket = uint32(12) // PACKET
)

const (
	ReservedOffset = 0 // 0
	ReservedSize   = 4

	FromOffset = ReservedOffset + ReservedSize // 4
	FromSize   = 4

	ToOffset = FromOffset + FromSize // 8
	ToSize   = 4

	IdOffset = ToOffset + ToSize // 12
	IdSize   = 8

	OperationOffset = IdOffset + IdSize // 20
	OperationSize   = 4

	ContentLengthOffset = OperationOffset + OperationSize // 24
	ContentLengthSize   = 8

	MessageSize = ContentLengthOffset + ContentLengthSize // 32
)

type Message struct {
	From          uint32 // 4 Bytes
	To            uint32 // 4 Bytes
	Id            uint64 // 8 Bytes
	Operation     uint32 // 4 Bytes
	ContentLength uint64 // 8 Bytes
}

type Packet struct {
	Message *Message
	Content *[]byte
}

type Handler struct{}

func NewDefaultHandler() Handler {
	return NewHandler()
}

func NewHandler() Handler {
	return Handler{}
}

func (handler *Handler) Encode(from uint32, to uint32, id uint64, operation uint32, contentLength uint64) ([MessageSize]byte, error) {
	return Encode(from, to, id, operation, contentLength)
}

func (handler *Handler) Decode(buf []byte) (message Message, err error) {
	return Decode(buf)
}

// Encode Message
func (fm *Message) Encode() (result [MessageSize]byte, err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.WithContext(recoveredErr.(error), ENCODE)
		}
	}()

	copy(result[ReservedOffset:ReservedOffset+ReservedSize], ReservedBytes)

	binary.BigEndian.PutUint32(result[FromOffset:FromOffset+FromSize], fm.From)
	binary.BigEndian.PutUint32(result[ToOffset:ToOffset+ToSize], fm.To)
	binary.BigEndian.PutUint64(result[IdOffset:IdOffset+IdSize], fm.Id)
	binary.BigEndian.PutUint32(result[OperationOffset:OperationOffset+OperationSize], fm.Operation)
	binary.BigEndian.PutUint64(result[ContentLengthOffset:ContentLengthOffset+ContentLengthSize], fm.ContentLength)

	return
}

// Decode Message
func (fm *Message) Decode(buf [MessageSize]byte) (err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.WithContext(recoveredErr.(error), DECODE)
		}
	}()

	if !bytes.Equal(buf[ReservedOffset:ReservedOffset+ReservedSize], ReservedBytes) {
		return InvalidMessageVersion
	}

	fm.From = binary.BigEndian.Uint32(buf[FromOffset : FromOffset+FromSize])
	fm.To = binary.BigEndian.Uint32(buf[ToOffset : ToOffset+ToSize])
	fm.Id = binary.BigEndian.Uint64(buf[IdOffset : IdOffset+IdSize])
	fm.Operation = binary.BigEndian.Uint32(buf[OperationOffset : OperationOffset+OperationSize])
	fm.ContentLength = binary.BigEndian.Uint64(buf[ContentLengthOffset : ContentLengthOffset+ContentLengthSize])

	return nil
}

// Encode without a Handler
func Encode(from uint32, to uint32, id uint64, operation uint32, contentLength uint64) ([MessageSize]byte, error) {
	message := Message{
		From:          from,
		To:            to,
		Id:            id,
		Operation:     operation,
		ContentLength: contentLength,
	}

	return message.Encode()
}

// Decode without a Handler
func Decode(buf []byte) (message Message, err error) {
	if len(buf) < MessageSize {
		return Message{}, InvalidBufferLength
	}

	err = message.Decode(*(*[MessageSize]byte)(unsafe.Pointer(&buf[0])))

	return
}

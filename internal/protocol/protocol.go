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
	"github.com/pkg/errors"
	"unsafe"
)

var (
	Encoding            = errors.New("error while encoding message")
	Decoding            = errors.New("error while decoding message")
	InvalidBufferLength = errors.New("invalid buffer length")
)

const (
	MessagePing   = uint16(10) // PING
	MessagePong   = uint16(11) // PONG
	MessagePacket = uint16(12) // PACKET
)

const (
	IdOffset = 0 // 0
	IdSize   = 2

	OperationOffset = IdOffset + IdSize // 2
	OperationSize   = 2

	ContentLengthOffset = OperationOffset + OperationSize // 4
	ContentLengthSize   = 4

	MessageSize = ContentLengthOffset + ContentLengthSize // 8
)

// Message is 8 bytes in length
type Message struct {
	Id            uint16 // 2 Bytes
	Operation     uint16 // 2 Bytes
	ContentLength uint32 // 4 Bytes
}

type Handler struct{}

func NewDefaultHandler() Handler {
	return NewHandler()
}

func NewHandler() Handler {
	return Handler{}
}

func (handler *Handler) Encode(id uint16, operation uint16, contentLength uint32) ([MessageSize]byte, error) {
	return Encode(id, operation, contentLength)
}

func (handler *Handler) Decode(buf []byte) (message Message, err error) {
	return Decode(buf)
}

// Encode Message
func (fm *Message) Encode() (result [MessageSize]byte, err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), Encoding.Error())
		}
	}()

	binary.BigEndian.PutUint16(result[IdOffset:IdOffset+IdSize], fm.Id)
	binary.BigEndian.PutUint16(result[OperationOffset:OperationOffset+OperationSize], fm.Operation)
	binary.BigEndian.PutUint32(result[ContentLengthOffset:ContentLengthOffset+ContentLengthSize], fm.ContentLength)

	return
}

// Decode Message
func (fm *Message) Decode(buf [MessageSize]byte) (err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), Decoding.Error())
		}
	}()

	fm.Id = binary.BigEndian.Uint16(buf[IdOffset : IdOffset+IdSize])
	fm.Operation = binary.BigEndian.Uint16(buf[OperationOffset : OperationOffset+OperationSize])
	fm.ContentLength = binary.BigEndian.Uint32(buf[ContentLengthOffset : ContentLengthOffset+ContentLengthSize])

	return nil
}

// Encode without a Handler
func Encode(id uint16, operation uint16, contentLength uint32) ([MessageSize]byte, error) {
	message := Message{
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

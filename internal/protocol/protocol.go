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

const (
	MessagePing   = uint32(0x0003) // PING
	MessagePong   = uint32(0x0004) // PONG
	MessagePacket = uint32(0x0005) // PACKET
)

const (
	ReservedV0Offset = 0
	ReservedV0Size   = 6

	VersionV0Offset = ReservedV0Offset + ReservedV0Size // 6
	VersionV0Size   = 2

	FromV0Offset = VersionV0Offset + VersionV0Size // 8
	FromV0Size   = 4

	ToV0Offset = FromV0Offset + FromV0Size // 12
	ToV0Size   = 4

	IdV0Offset = ToV0Offset + ToV0Size // 16
	IdV0Size   = 4

	OperationV0Offset = IdV0Offset + IdV0Size // 20
	OperationV0Size   = 4

	ContentLengthV0Offset = OperationV0Offset + OperationV0Size // 24
	ContentLengthV0Size   = 8

	MessageV0Size = ContentLengthV0Offset + ContentLengthV0Size // 32
	Version0      = uint16(0x00)                                // Version 0, 2 Bytes
)

type MessageV0 struct {
	From          uint32 // 4 Bytes
	To            uint32 // 4 Bytes
	Id            uint32 // 4 Bytes
	Operation     uint32 // 4 Bytes
	ContentLength uint64 // 8 Bytes
}

type PacketV0 struct {
	Message *MessageV0
	Content *[]byte
}

type V0Handler struct{}

func NewDefaultHandler() V0Handler {
	return NewV0Handler()
}

func NewV0Handler() V0Handler {
	return V0Handler{}
}

func (handler *V0Handler) Encode(from uint32, to uint32, id uint32, operation uint32, contentLength uint64) ([MessageV0Size]byte, error) {
	return EncodeV0(from, to, id, operation, contentLength)
}

func (handler *V0Handler) Decode(buf []byte) (message MessageV0, err error) {
	return DecodeV0(buf)
}

// Encode MessageV0
func (fm *MessageV0) Encode() (result [MessageV0Size]byte, err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.WithContext(recoveredErr.(error), ENCODE)
		}
	}()

	binary.BigEndian.PutUint16(result[VersionV0Offset:VersionV0Offset+VersionV0Size], Version0)
	binary.BigEndian.PutUint32(result[FromV0Offset:FromV0Offset+FromV0Size], fm.From)
	binary.BigEndian.PutUint32(result[ToV0Offset:ToV0Offset+ToV0Size], fm.To)
	binary.BigEndian.PutUint32(result[IdV0Offset:IdV0Offset+IdV0Size], fm.Id)
	binary.BigEndian.PutUint32(result[OperationV0Offset:OperationV0Offset+OperationV0Size], fm.Operation)
	binary.BigEndian.PutUint64(result[ContentLengthV0Offset:ContentLengthV0Offset+ContentLengthV0Size], fm.ContentLength)

	return
}

// Decode MessageV0
func (fm *MessageV0) Decode(buf [MessageV0Size]byte) (err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.WithContext(recoveredErr.(error), DECODE)
		}
	}()

	if !validVersion(binary.BigEndian.Uint16(buf[VersionV0Offset : VersionV0Offset+VersionV0Size])) {
		return InvalidMessageVersion
	}

	fm.From = binary.BigEndian.Uint32(buf[FromV0Offset : FromV0Offset+FromV0Size])
	fm.To = binary.BigEndian.Uint32(buf[ToV0Offset : ToV0Offset+ToV0Size])
	fm.Id = binary.BigEndian.Uint32(buf[IdV0Offset : IdV0Offset+IdV0Size])
	fm.Operation = binary.BigEndian.Uint32(buf[OperationV0Offset : OperationV0Offset+OperationV0Size])
	fm.ContentLength = binary.BigEndian.Uint64(buf[ContentLengthV0Offset : ContentLengthV0Offset+ContentLengthV0Size])

	return nil
}

// EncodeV0 without a Handler
func EncodeV0(from uint32, to uint32, id uint32, operation uint32, contentLength uint64) ([MessageV0Size]byte, error) {
	message := MessageV0{
		From:          from,
		To:            to,
		Id:            id,
		Operation:     operation,
		ContentLength: contentLength,
	}

	return message.Encode()
}

// DecodeV0 without a Handler
func DecodeV0(buf []byte) (message MessageV0, err error) {
	if len(buf) < MessageV0Size {
		return MessageV0{}, InvalidBufferLength
	}

	err = message.Decode(*(*[MessageV0Size]byte)(unsafe.Pointer(&buf[0])))

	return
}

func validVersion(version uint16) bool {
	switch version {
	case Version0:
		return true
	default:
		return false
	}
}

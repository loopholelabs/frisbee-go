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

package metadata

import (
	"encoding/binary"
	"unsafe"

	"github.com/pkg/errors"
)

var (
	Encoding            = errors.New("error while encoding metadata")
	Decoding            = errors.New("error while decoding metadata")
	InvalidBufferLength = errors.New("invalid buffer length")
)

const (
	PacketPing  = uint16(10) // PING
	PacketPong  = uint16(11) // PONG
	PacketProbe = uint16(12) // PACKET
)

const (
	IdOffset = 0 // 0
	IdSize   = 2

	OperationOffset = IdOffset + IdSize // 2
	OperationSize   = 2

	ContentLengthOffset = OperationOffset + OperationSize // 4
	ContentLengthSize   = 4

	Size = ContentLengthOffset + ContentLengthSize // 8
)

// Metadata is 8 bytes in length
type Metadata struct {
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

func (*Handler) Encode(id, operation uint16, contentLength uint32) ([Size]byte, error) {
	return Encode(id, operation, contentLength)
}

func (*Handler) Decode(buf []byte) (Metadata, error) {
	return Decode(buf)
}

// Encode Metadata
func (m *Metadata) Encode() (result [Size]byte, err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), Encoding.Error())
		}
	}()

	binary.BigEndian.PutUint16(result[IdOffset:IdOffset+IdSize], m.Id)
	binary.BigEndian.PutUint16(result[OperationOffset:OperationOffset+OperationSize], m.Operation)
	binary.BigEndian.PutUint32(result[ContentLengthOffset:ContentLengthOffset+ContentLengthSize], m.ContentLength)

	return
}

// Decode Metadata
func (m *Metadata) Decode(buf [Size]byte) (err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), Decoding.Error())
		}
	}()

	m.Id = binary.BigEndian.Uint16(buf[IdOffset : IdOffset+IdSize])
	m.Operation = binary.BigEndian.Uint16(buf[OperationOffset : OperationOffset+OperationSize])
	m.ContentLength = binary.BigEndian.Uint32(buf[ContentLengthOffset : ContentLengthOffset+ContentLengthSize])

	return nil
}

// Encode without a Handler
func Encode(id, operation uint16, contentLength uint32) ([Size]byte, error) {
	metadata := Metadata{
		Id:            id,
		Operation:     operation,
		ContentLength: contentLength,
	}

	return metadata.Encode()
}

// Decode without a Handler
func Decode(buf []byte) (metadata Metadata, err error) {
	if len(buf) < Size {
		return Metadata{}, InvalidBufferLength
	}

	err = metadata.Decode(*(*[Size]byte)(unsafe.Pointer(&buf[0])))

	return
}

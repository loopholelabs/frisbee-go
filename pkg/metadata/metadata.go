// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"encoding/binary"
	"errors"
	"unsafe"
)

var (
	EncodingErr            = errors.New("error while encoding metadata")
	DecodingErr            = errors.New("error while decoding metadata")
	InvalidBufferLengthErr = errors.New("invalid buffer length")
)

var (
	PacketMagicHeader = uint16(0x0F)
)

const (
	PacketPing  = uint16(10) // PING
	PacketPong  = uint16(11) // PONG
	PacketProbe = uint16(12) // PACKET
)

const (
	MagicOffset = 0 // 0
	MagicSize   = 2

	IdOffset = MagicOffset + MagicSize // 2
	IdSize   = 2

	OperationOffset = IdOffset + IdSize // 4
	OperationSize   = 2

	ContentLengthOffset = OperationOffset + OperationSize // 6
	ContentLengthSize   = 4

	Size = ContentLengthOffset + ContentLengthSize // 10
)

// Metadata is 8 bytes in length
type Metadata struct {
	Magic         uint16 // 2 bytes
	Id            uint16 // 2 Bytes
	Operation     uint16 // 2 Bytes
	ContentLength uint32 // 4 Bytes
}

// Encode Metadata
func (fm *Metadata) Encode() (b *Buffer, err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Join(recoveredErr.(error), EncodingErr)
		}
	}()

	b = NewBuffer()
	binary.BigEndian.PutUint16(b[MagicOffset:MagicOffset+MagicSize], fm.Magic)
	binary.BigEndian.PutUint16(b[IdOffset:IdOffset+IdSize], fm.Id)
	binary.BigEndian.PutUint16(b[OperationOffset:OperationOffset+OperationSize], fm.Operation)
	binary.BigEndian.PutUint32(b[ContentLengthOffset:ContentLengthOffset+ContentLengthSize], fm.ContentLength)

	return
}

// Decode Metadata
func (fm *Metadata) Decode(buf *Buffer) (err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Join(recoveredErr.(error), DecodingErr)
		}
	}()

	fm.Magic = binary.BigEndian.Uint16(buf[MagicOffset : MagicOffset+MagicSize])
	fm.Id = binary.BigEndian.Uint16(buf[IdOffset : IdOffset+IdSize])
	fm.Operation = binary.BigEndian.Uint16(buf[OperationOffset : OperationOffset+OperationSize])
	fm.ContentLength = binary.BigEndian.Uint32(buf[ContentLengthOffset : ContentLengthOffset+ContentLengthSize])

	return nil
}

func Encode(id, operation uint16, contentLength uint32) (*Buffer, error) {
	metadata := Metadata{
		Magic:         PacketMagicHeader,
		Id:            id,
		Operation:     operation,
		ContentLength: contentLength,
	}

	return metadata.Encode()
}

func Decode(buf []byte) (*Metadata, error) {
	if len(buf) < Size {
		return nil, InvalidBufferLengthErr
	}

	m := new(Metadata)
	return m, m.Decode((*Buffer)(unsafe.Pointer(&buf[0])))
}

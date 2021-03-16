package protocol

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"unsafe"
)

const (
	MessageHello   = uint16(0x0001) // HELLO
	MessageWelcome = uint16(0x0002) // WELCOME
	MessagePing    = uint16(0x0003) // PING
	MessagePong    = uint16(0x0004) // PONG
	MessagePacket  = uint16(0x0005) // PACKET
)

const (
	MagicBytes     = uint16(0x4642) // FBEE
	Version0       = uint8(0x01)    // Version 0
	HeaderLengthV0 = 16             // Version 0
)

type MessageV0 struct {
	Reserved      uint8  // 1 Byte
	MagicBytes    uint16 // 2 Bytes
	Version       uint8  // 1 Byte
	Id            uint16 // 2 Bytes
	Operation     uint16 // 2 Byte
	Routing       uint32 // 4 Bytes
	ContentLength uint32 // 4 Bytes
}

type V0Handler struct{}

func NewDefaultHandler() V0Handler {
	return NewV0Handler()
}

func NewV0Handler() V0Handler {
	return V0Handler{}
}

func (handler *V0Handler) Encode(id uint16, operation uint16, routing uint32, contentLength uint32) ([HeaderLengthV0]byte, error) {
	return EncodeV0(id, operation, routing, contentLength)
}

func (handler *V0Handler) Decode(buf []byte) (message MessageV0, err error) {
	return DecodeV0(buf)
}

// Encode MessageV0
func (fm *MessageV0) Encode() (result [HeaderLengthV0]byte, err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), "error encoding V0 message")
		}
	}()

	result[0] = byte(0x00) // Reserved
	binary.BigEndian.PutUint16(result[1:3], MagicBytes)
	result[3] = fm.Version
	binary.BigEndian.PutUint16(result[4:6], fm.Id)
	binary.BigEndian.PutUint16(result[6:8], fm.Operation)
	binary.BigEndian.PutUint32(result[8:12], fm.Routing)
	binary.BigEndian.PutUint32(result[12:16], fm.ContentLength)

	return
}

// Decode MessageV0
func (fm *MessageV0) Decode(buf [HeaderLengthV0]byte) (err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), "error decoding V0 message")
		}
	}()

	fm.Version = buf[3]
	if !validVersion(fm.Version) {
		return errors.New("invalid message version")
	}
	fm.Id = binary.BigEndian.Uint16(buf[4:6])
	fm.Operation = binary.BigEndian.Uint16(buf[6:8])
	fm.Routing = binary.BigEndian.Uint32(buf[8:12])
	fm.ContentLength = binary.BigEndian.Uint32(buf[12:16])

	return nil
}

// EncodeV0 without a Handler
func EncodeV0(id uint16, operation uint16, routing uint32, contentLength uint32) ([HeaderLengthV0]byte, error) {
	message := MessageV0{
		Version:       Version0,
		Id:            id,
		Operation:     operation,
		Routing:       routing,
		ContentLength: contentLength,
	}

	return message.Encode()
}

// DecodeV0 without a Handler
func DecodeV0(buf []byte) (message MessageV0, err error) {
	if len(buf) < HeaderLengthV0 {
		return MessageV0{}, errors.New("invalid buffer length")
	}

	err = message.Decode(*(*[HeaderLengthV0]byte)(unsafe.Pointer(&buf[0])))

	return
}

func validVersion(version uint8) bool {
	switch version {
	case Version0:
		return true
	default:
		return false
	}
}

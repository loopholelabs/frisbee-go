package protocol

import (
	"encoding/binary"
	"github.com/pkg/errors"
)

const (
	MessageHello   = uint16(0x0001) // HELLO
	MessageWelcome = uint16(0x0002) // WELCOME
	MessagePing    = uint16(0x0003) // PING
	MessagePong    = uint16(0x0004) // PONG
	MessagePacket  = uint16(0x0005) // PACKET
)

const (
	MagicBytes     = uint32(0x46424545) // FBEE
	Version0       = uint8(0x01)        // Version 0
	HeaderLengthV0 = 16                 // Version 0
)

type MessageV0 struct {
	Reserved      uint8  // 1 Byte
	MagicBytes    uint32 // 4 Bytes
	Version       uint8  // 1 Byte
	Operation     uint16 // 2 Byte
	Routing       uint32 // 4 Bytes
	ContentLength uint32 // 4 Bytes
	Content       []byte
}

type V0Handler struct{}

func NewDefaultHandler() V0Handler {
	return NewV0Handler()
}

func NewV0Handler() V0Handler {
	return V0Handler{}
}

func (handler *V0Handler) Encode(operation uint16, routing uint32, buf []byte) ([]byte, error) {
	message := MessageV0{
		Version:       Version0,
		Operation:     operation,
		Routing:       routing,
		ContentLength: uint32(len(buf)),
		Content:       buf,
	}

	return message.Encode()
}

func (handler *V0Handler) Decode(buf []byte) (message MessageV0, err error) {
	message = MessageV0{
		Content: buf,
	}
	err = message.Decode(false)

	return
}

// Encode MessageV0
func (fm *MessageV0) Encode() (result []byte, err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), "Error Encoding V0 Message")
		}
	}()

	intermediate := make([]uint8, HeaderLengthV0)
	intermediate[0] = uint8(0x00) // Reserved
	intermediate[1] = uint8(0x46) // F
	intermediate[2] = uint8(0x42) // B
	intermediate[3] = uint8(0x45) // E
	intermediate[4] = uint8(0x45) // E
	intermediate[5] = fm.Version
	binary.BigEndian.PutUint16(intermediate[6:8], fm.Operation)
	binary.BigEndian.PutUint32(intermediate[8:12], fm.Routing)
	binary.BigEndian.PutUint32(intermediate[12:16], fm.ContentLength)

	result = make([]byte, HeaderLengthV0+fm.ContentLength)
	copy(result[:HeaderLengthV0], intermediate)

	if fm.ContentLength > 0 {
		copy(result[HeaderLengthV0:], fm.Content)
	}

	return
}

// Decode MessageV0
func (fm *MessageV0) Decode(headerOnly bool) (err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), "Error Decoding V0 Message")
		}
	}()

	if len(fm.Content) < HeaderLengthV0 {
		return errors.New("Invalid Message Header (Too Small)")
	}

	fm.Version = fm.Content[5]
	if !validVersion(fm.Version) {
		return errors.New("Invalid Message Version")
	}

	fm.Operation = binary.BigEndian.Uint16(fm.Content[6:8])
	fm.Routing = binary.BigEndian.Uint32(fm.Content[8:12])
	fm.ContentLength = binary.BigEndian.Uint32(fm.Content[12:16])

	if fm.ContentLength > 0 && !headerOnly {
		fm.Content = fm.Content[HeaderLengthV0:]
		if uint32(len(fm.Content)) != fm.ContentLength {
			return errors.New("Invalid Data is not the same length as Data Length")
		}
	} else if fm.ContentLength < 1 {
		fm.Content = make([]byte, 0)
	}

	return nil
}

// EncodeV0 without a Handler
func EncodeV0(operation uint16, routing uint32, buf []byte) ([]byte, error) {
	message := MessageV0{
		Version:       Version0,
		Operation:     operation,
		Routing:       routing,
		ContentLength: uint32(len(buf)),
		Content:       buf,
	}

	return message.Encode()
}

// DecodeV0 without a Handler
func DecodeV0(buf []byte, headerOnly bool) (message MessageV0, err error) {
	message = MessageV0{
		Content: buf,
	}

	return message, message.Decode(headerOnly)
}

func validVersion(version uint8) bool {
	switch version {
	case Version0:
		return true
	default:
		return false
	}
}

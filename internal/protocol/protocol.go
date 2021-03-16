package protocol

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"io"
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
}

type V0Handler struct{}

func NewDefaultHandler() V0Handler {
	return NewV0Handler()
}

func NewV0Handler() V0Handler {
	return V0Handler{}
}

func (handler *V0Handler) Encode(operation uint16, routing uint32, contentLength uint32) ([HeaderLengthV0]byte, error) {
	message := MessageV0{
		Version:       Version0,
		Operation:     operation,
		Routing:       routing,
		ContentLength: contentLength,
	}

	return message.Encode()
}

func (handler *V0Handler) Decode(buf []byte) (message MessageV0, err error) {
	if len(buf) < HeaderLengthV0 {
		return MessageV0{}, errors.New("Invalid Buffer Length")
	}

	err = message.Decode(*(*[HeaderLengthV0]byte)(unsafe.Pointer(&buf[0])))

	return
}

func (handler *V0Handler) Write(message [HeaderLengthV0]byte, content *[]byte, destination io.Writer) (n int, err error) {
	n, err = destination.Write(message[:])
	m, err := destination.Write(*content)
	n += m

	return
}

// Encode MessageV0
func (fm *MessageV0) Encode() (result [HeaderLengthV0]byte, err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), "Error Encoding V0 Message")
		}
	}()

	result[0] = byte(0x00) // Reserved
	result[1] = byte(0x46) // F
	result[2] = byte(0x42) // B
	result[3] = byte(0x45) // E
	result[4] = byte(0x45) // E
	result[5] = fm.Version
	binary.BigEndian.PutUint16(result[6:8], fm.Operation)
	binary.BigEndian.PutUint32(result[8:12], fm.Routing)
	binary.BigEndian.PutUint32(result[12:16], fm.ContentLength)

	return
}

// Decode MessageV0
func (fm *MessageV0) Decode(buf [HeaderLengthV0]byte) (err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), "Error Decoding V0 Message")
		}
	}()

	fm.Version = buf[5]
	if !validVersion(fm.Version) {
		return errors.New("Invalid Message Version")
	}

	fm.Operation = binary.BigEndian.Uint16(buf[6:8])
	fm.Routing = binary.BigEndian.Uint32(buf[8:12])
	fm.ContentLength = binary.BigEndian.Uint32(buf[12:16])

	return nil
}

// EncodeV0 without a Handler
func EncodeV0(operation uint16, routing uint32, contentLength uint32) ([HeaderLengthV0]byte, error) {
	message := MessageV0{
		Version:       Version0,
		Operation:     operation,
		Routing:       routing,
		ContentLength: contentLength,
	}

	return message.Encode()
}

func Write(message [HeaderLengthV0]byte, content *[]byte, destination io.Writer) (n int, err error) {
	n, err = destination.Write(message[:])
	m, err := destination.Write(*content)
	n += m

	return
}

// DecodeV0 without a Handler
func DecodeV0(buf []byte) (message MessageV0, err error) {
	if len(buf) < HeaderLengthV0 {
		return MessageV0{}, errors.New("Invalid Buffer Length")
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

package protocol

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
)

// Various Message Types and Operations
const (
	MessageHello   = uint16(0x0001) // HELLO
	MessageWelcome = uint16(0x0002) // WELCOME
	MessagePing    = uint16(0x0003) // PING
	MessagePong    = uint16(0x0004) // PONG
	MessagePacket  = uint16(0x0005) // PACKET
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

const (
	MagicBytes     = uint32(0x46424545) // FBEE
	Version0       = uint8(0x01)        // Version 0
	HeaderLengthV0 = 16                 // Version 0
)

type V0Handler struct {
	Unsafe bool
}

func NewDefaultHandler() *V0Handler {
	return NewV0Handler(false)
}

func NewV0Handler(unsafe bool) *V0Handler {
	return &V0Handler{
		Unsafe: unsafe,
	}
}

func (handler *V0Handler) setUnsafe(unsafe bool) {
	handler.Unsafe = unsafe
}

func (handler *V0Handler) Encode(operation uint16, routing uint32, buf []byte) (data []byte, err error) {
	if !validOperation(operation) {
		err = errors.New("Invalid Message Operation")
		return
	}

	message := MessageV0{
		Version:       Version0,
		Operation:     operation,
		Routing:       routing,
		ContentLength: uint32(len(buf)),
		Content:       buf,
	}

	switch handler.Unsafe {
	case true:
		return message.UnsafeEncode()
	default:
		return message.Encode()
	}
}

func (handler *V0Handler) Decode(buf []byte) (message MessageV0, err error) {

	message = MessageV0{
		Content: buf,
	}

	switch handler.Unsafe {
	case true:
		err = message.UnsafeDecode(false)
	default:
		err = message.Decode(false)
	}

	return
}

// UnsafeEncode MessageV0
func (fm *MessageV0) UnsafeEncode() (result []byte, err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), "Error Encoding Message")
		}
	}()

	result = make([]byte, HeaderLengthV0+fm.ContentLength)
	result[0] = uint8(0x00) // Reserved
	result[1] = uint8(0x46) // F
	result[2] = uint8(0x42) // B
	result[3] = uint8(0x45) // E
	result[4] = uint8(0x45) // E
	result[5] = fm.Version
	binary.BigEndian.PutUint16(result[6:8], fm.Operation)
	binary.BigEndian.PutUint32(result[8:12], fm.Routing)
	binary.BigEndian.PutUint32(result[12:16], fm.ContentLength)

	if fm.ContentLength > 0 {
		copy(result[HeaderLengthV0:], fm.Content)
	}

	return
}

// Encode MessageV0
func (fm *MessageV0) Encode() ([]byte, error) {
	var result []byte
	buffer := bytes.NewBuffer(result)

	if err := binary.Write(buffer, binary.BigEndian, uint8(0x00)); err != nil {
		return nil, errors.Wrap(err, "Unable to pack Reserved Bytes")
	}

	if err := binary.Write(buffer, binary.BigEndian, MagicBytes); err != nil {
		return nil, errors.Wrap(err, "Unable to pack Magic Bytes")
	}

	if err := binary.Write(buffer, binary.BigEndian, fm.Version); err != nil {
		return nil, errors.Wrap(err, "Unable to pack Message Version")
	}

	if err := binary.Write(buffer, binary.BigEndian, fm.Operation); err != nil {
		return nil, errors.Wrap(err, "Unable to pack Message Operation")
	}
	if err := binary.Write(buffer, binary.BigEndian, fm.Routing); err != nil {
		return nil, errors.Wrap(err, "Unable to pack Message Routing")
	}
	if err := binary.Write(buffer, binary.BigEndian, fm.ContentLength); err != nil {
		return nil, errors.Wrap(err, "Unable to pack Message Content Length")
	}

	if fm.ContentLength > 0 {
		if err := binary.Write(buffer, binary.BigEndian, fm.Content); err != nil {
			return nil, errors.Wrap(err, "Unable to pack Message Content")
		}
	}

	return buffer.Bytes(), nil
}

// UnsafeDecode MessageV0
func (fm *MessageV0) UnsafeDecode(headerOnly bool) (err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), "Error Unsafe Decoding Message")
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
	if !validOperation(fm.Operation) {
		return errors.New("Invalid Message Operation")
	}

	fm.Routing = binary.BigEndian.Uint32(fm.Content[8:12])
	fm.ContentLength = binary.BigEndian.Uint32(fm.Content[12:16])

	if fm.ContentLength > 0 && !headerOnly {
		fm.Content = fm.Content[HeaderLengthV0:]
		if uint32(len(fm.Content)) != fm.ContentLength {
			return errors.New("Invalid Data is not the same length as Data Length")
		}
	}

	return nil
}

// Decode MessageV0
func (fm *MessageV0) Decode(headerOnly bool) (err error) {

	if len(fm.Content) < HeaderLengthV0 {
		return errors.New("Invalid Message Header (Too Small)")
	}

	byteBuffer := bytes.NewBuffer(fm.Content[5:HeaderLengthV0])

	if err = binary.Read(byteBuffer, binary.BigEndian, &fm.Version); err != nil {
		return errors.Wrap(err, "Unable to read Version")
	}

	if !validVersion(fm.Version) {
		return errors.New("Invalid Message Version")
	}

	if err = binary.Read(byteBuffer, binary.BigEndian, &fm.Operation); err != nil {
		return errors.Wrap(err, "Unable to read Message Operation")
	}

	if !validOperation(fm.Operation) {
		return errors.New("Invalid Message Type")
	}

	if err = binary.Read(byteBuffer, binary.BigEndian, &fm.Routing); err != nil {
		return errors.Wrap(err, "Unable to read Message Routing")
	}

	if err = binary.Read(byteBuffer, binary.BigEndian, &fm.ContentLength); err != nil {
		return errors.Wrap(err, "Unable to read Data Length")
	}

	if fm.ContentLength > 0 && !headerOnly {
		fm.Content = fm.Content[HeaderLengthV0:]
		if uint32(len(fm.Content)) != fm.ContentLength {
			return errors.New("Invalid Data is not the same length as Data Length")
		}
	}

	return nil
}

// EncodeV0 without a Handler
func EncodeV0(operation uint16, routing uint32, buf []byte, unsafe bool) (data []byte, err error) {
	message := MessageV0{
		Version:       Version0,
		Operation:     operation,
		Routing:       routing,
		ContentLength: uint32(len(buf)),
		Content:       buf,
	}

	switch unsafe {
	case true:
		return message.UnsafeEncode()
	default:
		return message.Encode()
	}
}

// DecodeV0 without a Handler
func DecodeV0(buf []byte, unsafe bool, headerOnly bool) (message MessageV0, err error) {
	message = MessageV0{
		Content: buf,
	}

	switch unsafe {
	case true:
		err = message.UnsafeDecode(headerOnly)
	default:
		err = message.Decode(headerOnly)
	}

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

func validOperation(operation uint16) bool {
	switch operation {
	case MessageHello, MessageWelcome, MessagePing, MessagePong, MessagePacket:
		return true
	default:
		return false
	}
}

package protocol

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
)

const (
	MessagePing    = uint16(0x0001) // PING
	MessagePong    = uint16(0x0002) // PONG
	MessageHello   = uint16(0x0011) // HELLO
	MessageWelcome = uint16(0x0012) // WELCOME
)

// MessageV0 Header (Version + MessageType + DataLength) is 8 Bytes
type MessageV0 struct {
	Version     uint16 // 2 Bytes
	MessageType uint16 // 2 Bytes
	DataLength  uint32 // 4 Bytes
	MessageData []byte
}

const (
	Version0               = uint16(0x0FD1) // Version 0
	HeaderLengthV0         = 8              // Version 0
	DefaultProtocolVersion = Version0       // Version 0
)

type MessageOptions struct {
	Version uint16
	Unsafe  bool
}

type MessageHandler struct {
	Options *MessageOptions
}

type Message interface {
	Encode() []byte
	Decode() error
	SafeEncode() ([]byte, error)
	SafeDecode() error
	Ver() uint16
	Type() uint16
	Length() uint32
	Data() ([]byte, error)
}

func DefaultOptions() *MessageOptions {
	return &MessageOptions{
		Version: DefaultProtocolVersion,
		Unsafe:  false,
	}
}

func NewMessageHandler(options *MessageOptions) (*MessageHandler, error) {
	if !validVersion(options.Version) {
		return nil, errors.New("Invalid Message Options (Version)")
	}

	return &MessageHandler{
		Options: options,
	}, nil
}

func (handler *MessageHandler) Encode(messageType uint16, buf []byte) (data []byte, err error) {
	if !validMessageType(messageType) {
		err = errors.New("Invalid Message Type")
		return
	}

	var message Message

	switch handler.Options.Version {
	default:
		// Assume Default Version
		message = &MessageV0{
			Version:     Version0,
			MessageType: messageType,
			DataLength:  uint32(len(buf)),
			MessageData: buf,
		}
	}

	switch handler.Options.Unsafe {
	case true:
		defer func() {
			if recoveredErr := recover(); recoveredErr != nil {
				err = errors.Wrap(recoveredErr.(error), "Error Encoding Message")
			}
		}()

		data = message.Encode()
		return
	default:
		return message.SafeEncode()
	}
}

func (handler *MessageHandler) Decode(buf []byte) (message Message, err error) {

	switch handler.Options.Version {
	default:
		// Assume Default Version
		message = &MessageV0{
			MessageData: buf,
		}
	}

	switch handler.Options.Unsafe {
	case true:
		defer func() {
			if recoveredErr := recover(); recoveredErr != nil {
				err = errors.Wrap(recoveredErr.(error), "Error Unsafe Decoding Message")
			}
		}()
		err = message.Decode()
	default:
		err = message.SafeDecode()
	}
	return

}

// Encode MessageV0
func (fm *MessageV0) Encode() (result []byte) {
	intermediate := make([]byte, 2)

	binary.BigEndian.PutUint16(intermediate, fm.Version)
	result = append(result, intermediate...)

	binary.BigEndian.PutUint16(intermediate, fm.MessageType)
	result = append(result, intermediate...)

	intermediate = make([]byte, 4)
	binary.BigEndian.PutUint32(intermediate, fm.DataLength)
	result = append(result, intermediate...)

	if fm.DataLength > 0 {
		result = append(result, fm.MessageData...)
	}
	return
}

// SafeEncode MessageV0
func (fm *MessageV0) SafeEncode() ([]byte, error) {
	result := make([]byte, 0)
	buffer := bytes.NewBuffer(result)

	if err := binary.Write(buffer, binary.BigEndian, fm.Version); err != nil {
		return nil, errors.Wrap(err, "Unable to pack Message Version")
	}

	if err := binary.Write(buffer, binary.BigEndian, fm.MessageType); err != nil {
		return nil, errors.Wrap(err, "Unable to pack Message Type")
	}
	if err := binary.Write(buffer, binary.BigEndian, fm.DataLength); err != nil {
		return nil, errors.Wrap(err, "Unable to pack Data Length")
	}
	if fm.DataLength > 0 {
		if err := binary.Write(buffer, binary.BigEndian, fm.MessageData); err != nil {
			return nil, errors.Wrap(err, "Unable to pack Message Data")
		}
	}

	return buffer.Bytes(), nil
}

// Decode MessageV0
func (fm *MessageV0) Decode() error {
	if len(fm.MessageData) < HeaderLengthV0 {
		return errors.New("Invalid Message Header (Too Small)")
	}

	version := binary.BigEndian.Uint16(fm.MessageData[0:2])
	if !validVersion(version) {
		return errors.New("Invalid Message Version")
	}

	messageType := binary.BigEndian.Uint16(fm.MessageData[2:4])
	if !validMessageType(messageType) {
		return errors.New("Invalid Message Type")
	}

	dataLength := binary.BigEndian.Uint32(fm.MessageData[4:8])

	fm.Version = version
	fm.MessageType = messageType
	fm.DataLength = dataLength

	if dataLength > 0 {
		data := fm.MessageData[HeaderLengthV0:]
		if uint32(len(data)) != dataLength {
			return errors.New("Invalid Data is not the same length as Data Length")
		}
		fm.MessageData = data
	}

	return nil
}

// SafeDecode MessageV0
func (fm *MessageV0) SafeDecode() error {

	if len(fm.MessageData) < HeaderLengthV0 {
		return errors.New("Invalid Message Header (Too Small)")
	}

	byteBuffer := bytes.NewBuffer(fm.MessageData[:HeaderLengthV0])

	var err error
	var version, messageType uint16
	var dataLength uint32

	err = binary.Read(byteBuffer, binary.BigEndian, &version)
	if err != nil {
		return errors.Wrap(err, "Unable to read Version")
	}

	if !validVersion(version) {
		return errors.New("Invalid Message Version")
	}

	err = binary.Read(byteBuffer, binary.BigEndian, &messageType)
	if err != nil {
		return errors.Wrap(err, "Unable to read Message Type")
	}

	if !validMessageType(messageType) {
		return errors.New("Invalid Message Type")
	}

	err = binary.Read(byteBuffer, binary.BigEndian, &dataLength)
	if err != nil {
		return errors.Wrap(err, "Unable to read Data Length")
	}

	fm.Version = version
	fm.MessageType = messageType
	fm.DataLength = dataLength

	if dataLength > 0 {
		data := fm.MessageData[HeaderLengthV0:]
		if uint32(len(data)) != dataLength {
			return errors.New("Invalid Data is not the same length as Data Length")
		}
		fm.MessageData = data
	}

	return nil
}

func (fm *MessageV0) Ver() uint16 {
	return fm.Version
}

func (fm *MessageV0) Type() uint16 {
	return fm.MessageType
}

func (fm *MessageV0) Length() uint32 {
	return fm.DataLength
}

func (fm *MessageV0) Data() ([]byte, error) {
	if fm.DataLength > 0 {
		return fm.MessageData, nil
	}

	return nil, errors.New("Message contains no data")
}

func validVersion(version uint16) bool {
	switch version {
	case Version0:
		return true
	default:
		return false
	}
}

func validMessageType(messageType uint16) bool {
	switch messageType {
	case MessagePing, MessagePong, MessageHello, MessageWelcome:
		return true
	default:
		return false
	}
}

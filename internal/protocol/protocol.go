package protocol

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"unsafe"
)

const (
	MessagePing   = uint32(0x0003) // PING
	MessagePong   = uint32(0x0004) // PONG
	MessagePacket = uint32(0x0005) // PACKET
)

const (
	Version0       = uint16(0x01) // Version 0
	HeaderLengthV0 = 32           // Version 0
)

type MessageV0 struct {
	From          uint32 // 4 Bytes
	To            uint32 // 4 Bytes
	Id            uint32 // 4 Bytes
	Operation     uint32 // 4 Bytes
	ContentLength uint64 // 8 Bytes
}

/*
0:  0x00 // Reserved
1:  0x00 // Reserved
2:  0x00 // Reserved
3:  0x00 // Reserved
4:  0x00 // Reserved
5:  0x00 // Reserved

6:  0x00 // Version0[0]
7:  0x01 // Version0[1]

8:  0x00 // From[0]
9:  0x00 // From[1]
10: 0x00 // From[2]
11: 0x00 // From[3]

12: 0x00 // To[0]
13: 0x00 // To[1]
14: 0x00 // To[2]
15: 0x00 // To[3]

16: 0x00 // Id[0]
17: 0x00 // Id[1]
18: 0x00 // Id[2]
19: 0x00 // Id[3]

20: 0x00 // Operation[0]
21: 0x00 // Operation[1]
22: 0x00 // Operation[2]
23: 0x00 // Operation[3]

24: 0x00 // ContentLength[0]
25: 0x00 // ContentLength[1]
26: 0x00 // ContentLength[2]
27: 0x00 // ContentLength[3]
28: 0x00 // ContentLength[4]
29: 0x00 // ContentLength[5]
30: 0x00 // ContentLength[6]
31: 0x00 // ContentLength[7]
*/

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

func (handler *V0Handler) Encode(from uint32, to uint32, id uint32, operation uint32, contentLength uint64) ([HeaderLengthV0]byte, error) {
	return EncodeV0(from, to, id, operation, contentLength)
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

	binary.BigEndian.PutUint16(result[6:8], Version0)
	binary.BigEndian.PutUint32(result[8:12], fm.From)
	binary.BigEndian.PutUint32(result[12:16], fm.To)
	binary.BigEndian.PutUint32(result[16:20], fm.Id)
	binary.BigEndian.PutUint32(result[20:24], fm.Operation)
	binary.BigEndian.PutUint64(result[24:32], fm.ContentLength)

	return
}

// Decode MessageV0
func (fm *MessageV0) Decode(buf [HeaderLengthV0]byte) (err error) {
	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = errors.Wrap(recoveredErr.(error), "error decoding V0 message")
		}
	}()

	if !validVersion(binary.BigEndian.Uint16(buf[6:8])) {
		return errors.New("invalid message version")
	}

	fm.From = binary.BigEndian.Uint32(buf[8:12])
	fm.To = binary.BigEndian.Uint32(buf[12:16])
	fm.Id = binary.BigEndian.Uint32(buf[16:20])
	fm.Operation = binary.BigEndian.Uint32(buf[20:24])
	fm.ContentLength = binary.BigEndian.Uint64(buf[24:32])

	return nil
}

// EncodeV0 without a Handler
func EncodeV0(from uint32, to uint32, id uint32, operation uint32, contentLength uint64) ([HeaderLengthV0]byte, error) {
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
	if len(buf) < HeaderLengthV0 {
		return MessageV0{}, errors.New("invalid buffer length")
	}

	err = message.Decode(*(*[HeaderLengthV0]byte)(unsafe.Pointer(&buf[0])))

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

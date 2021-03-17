package codec

import (
	"encoding/binary"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/panjf2000/gnet"
	"github.com/pkg/errors"
)

type Packet struct {
	Message *protocol.MessageV0
	Content *[]byte
}

type ICodec struct {
	Packets map[uint16]*Packet
}

// Encode for gnet codec
func (codec *ICodec) Encode(_ gnet.Conn, buf []byte) ([]byte, error) {
	return buf, nil
}

// Encode for gnet codec
func (codec *ICodec) Decode(c gnet.Conn) ([]byte, error) {
	if size, message := c.ReadN(protocol.HeaderLengthV0); size == protocol.HeaderLengthV0 {
		decodedMessage, err := protocol.DecodeV0(message)
		if err != nil {
			c.ResetBuffer()
			return nil, errors.Wrap(err, "error decoding header")
		}
		key := [2]byte{}
		binary.BigEndian.PutUint16(key[:], decodedMessage.Id)
		packet := &Packet{
			Message: &decodedMessage,
		}
		if decodedMessage.ContentLength > 0 {
			c.ShiftN(protocol.HeaderLengthV0)
			if contentLength, content := c.ReadN(int(decodedMessage.ContentLength)); contentLength == int(decodedMessage.ContentLength) {
				packet.Content = &content
				codec.Packets[decodedMessage.Id] = packet
				return key[:], nil
			}
			return nil, errors.New("invalid content length")
		}
		codec.Packets[decodedMessage.Id] = packet
		return key[:], nil
	}
	return nil, errors.New("invalid message length")
}

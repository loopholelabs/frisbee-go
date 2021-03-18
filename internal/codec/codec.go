package codec

import (
	"encoding/binary"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/panjf2000/gnet"
	"github.com/pkg/errors"
)

type Packet struct {
	Message *protocol.MessageV0
	Content []byte
}

type ICodec struct {
	Packets map[uint32]*Packet
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
		key := [4]byte{}
		binary.BigEndian.PutUint32(key[:], decodedMessage.Id)
		packet := &Packet{
			Message: &decodedMessage,
		}
		if decodedMessage.ContentLength > 0 {
			if contentLength, content := c.ReadN(int(decodedMessage.ContentLength + protocol.HeaderLengthV0)); contentLength == int(decodedMessage.ContentLength+protocol.HeaderLengthV0) {
				c.ShiftN(int(decodedMessage.ContentLength + protocol.HeaderLengthV0))
				packet.Content = content[protocol.HeaderLengthV0:]
				codec.Packets[decodedMessage.Id] = packet
				return key[:], nil
			}
			return nil, errors.New("invalid content length")
		}
		c.ShiftN(protocol.HeaderLengthV0)
		codec.Packets[decodedMessage.Id] = packet
		return key[:], nil
	}
	return nil, errors.New("invalid message length")
}

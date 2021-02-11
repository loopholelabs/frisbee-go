package codec

import (
	"errors"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/panjf2000/gnet"
	"log"
)

type MessageV0 struct {
	Operation uint16
	Routing   uint32
}

// Encode for gnet codec
func (fb *MessageV0) Encode(c gnet.Conn, buf []byte) ([]byte, error) {
	config := c.Context().(MessageV0)
	return protocol.EncodeV0(config.Operation, config.Routing, buf, true)
}

// Encode for gnet codec
func (fb *MessageV0) Decode(c gnet.Conn) ([]byte, error) {
	if size, encodedHeader := c.ReadN(protocol.HeaderLengthV0); size == protocol.HeaderLengthV0 {
		decodedHeader, err := protocol.DecodeV0(encodedHeader, true, true)
		if err != nil {
			c.ResetBuffer()
			log.Printf("Error decoding header: %s", err.Error())
			return nil, err
		}

		if contentLength, content := c.ReadN(int(decodedHeader.ContentLength + protocol.HeaderLengthV0)); contentLength == int(decodedHeader.ContentLength+protocol.HeaderLengthV0) {
			c.ShiftN(int(decodedHeader.ContentLength + protocol.HeaderLengthV0))
			return content[protocol.HeaderLengthV0:], nil
		}

		return nil, errors.New("not enough content")
	}

	return nil, errors.New("not enough data")
}

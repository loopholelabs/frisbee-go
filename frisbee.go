package frisbee

import (
	"github.com/loophole-labs/frisbee/internal/conn"
	"github.com/loophole-labs/frisbee/internal/protocol"
)

type Message protocol.MessageV0
type ServerRouteFunc func(c Conn, incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)
type ServerRouter map[uint16]ServerRouteFunc
type ClientRouteFunc func(incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)
type ClientRouter map[uint16]ClientRouteFunc
type Action int
type Conn struct {
	conn.Conn
}

const None = Action(0)
const Close = Action(1)
const Shutdown = Action(2)

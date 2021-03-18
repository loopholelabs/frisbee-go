package frisbee

import (
	"github.com/loophole-labs/frisbee/internal/protocol"
)

type Message protocol.MessageV0
type RouteFunc func(incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)
type Router map[uint16]RouteFunc
type Action int

const None = Action(0)
const Close = Action(1)
const Shutdown = Action(2)

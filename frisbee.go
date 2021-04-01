package frisbee

import (
	"github.com/loophole-labs/frisbee/internal/protocol"
)

type Action int
type Message protocol.MessageV0

const (
	None = Action(iota)
	Close
	Shutdown
)

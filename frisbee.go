package frisbee

import (
	"github.com/loophole-labs/frisbee/internal/errors"
	"github.com/loophole-labs/frisbee/internal/protocol"
)

const (
	DIAL   errors.ErrorContext = "error while dialing connection"
	CLOSE  errors.ErrorContext = "error while closing connection"
	WRITE  errors.ErrorContext = "error while writing to buffer"
	PUSH   errors.ErrorContext = "error while pushing packet to message queue"
	POP    errors.ErrorContext = "error while popping packet from message queue"
	ACCEPT errors.ErrorContext = "error while accepting connections"
)

var (
	InvalidContentLength     = errors.New("invalid content length")
	ConnectionClosed         = errors.New("connection closed")
	ConnectionPaused         = errors.New("connection paused")
	ConnectionNotInitialized = errors.New("connection not initialized")
	InvalidBufferContents    = errors.New("invalid buffer contents")
	InvalidBufferLength      = errors.New("invalid buffer length")
)

type Action int
type Message protocol.MessageV0

const (
	None = Action(iota)
	Close
	Shutdown
)

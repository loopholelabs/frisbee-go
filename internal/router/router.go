package router

import (
	"github.com/loophole-labs/frisbee/internal/protocol"
)

type Handler interface {
}

type HandlerV0 struct {
	MessageMap map[uint16]func(message protocol.MessageV0) error
}

func (handler HandlerV0) Route(message protocol.MessageV0) error {
	return handler.MessageMap[message.Operation](message)
}

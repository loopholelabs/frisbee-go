package server

import (
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewServer(t *testing.T) {
	addr := "tcp://123.0.0.0:123"
	router := make(frisbee.Router)
	router[protocol.MessagePing] = func(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	server := NewServer(addr, router, WithAsync(true))
	assert.Equal(t, router, server.router)
	assert.Equal(t, loadOptions(WithAsync(true)), server.options)
	assert.Equal(t, addr, server.addr)
}

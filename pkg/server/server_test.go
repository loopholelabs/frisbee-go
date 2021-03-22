package server

import (
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewServer(t *testing.T) {
	addr := "tcp://:8192"
	router := make(frisbee.ServerRouter)
	router[protocol.MessagePing] = func(_ frisbee.Conn, _ frisbee.Message, _ []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	server := NewServer(addr, router, WithAsync(true))
	assert.Equal(t, router, server.router)
	assert.Equal(t, LoadOptions(WithAsync(true)), server.Options)
	assert.Equal(t, addr, server.addr)
}

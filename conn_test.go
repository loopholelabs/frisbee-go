package frisbee

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewIncomingBuffer(t *testing.T) {
	newBuffer := newIncomingBuffer()
	assert.Equal(t, DefaultBufferSize, newBuffer.buffer.Cap())
}

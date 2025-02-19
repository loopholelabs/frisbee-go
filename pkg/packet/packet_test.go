// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	p := Get()

	assert.IsType(t, new(Packet), p)
	assert.NotNil(t, p.Metadata)
	assert.Equal(t, uint16(0), p.Metadata.Magic)
	assert.Equal(t, uint16(0), p.Metadata.Id)
	assert.Equal(t, uint16(0), p.Metadata.Operation)
	assert.Equal(t, uint32(0), p.Metadata.ContentLength)
	assert.Equal(t, Get().Content, p.Content)

	Put(p)
}

func TestWrite(t *testing.T) {
	t.Parallel()

	p := Get()

	b := make([]byte, 32)
	_, err := rand.Read(b)
	assert.NoError(t, err)

	p.Content.Write(b)
	assert.Equal(t, b, p.Content.Bytes())

	p.Reset()
	assert.NotEqual(t, b, p.Content.Bytes())
	assert.Equal(t, 0, p.Content.Len())
	assert.Equal(t, 512, p.Content.Cap())

	b = make([]byte, 1024)
	_, err = rand.Read(b)
	assert.NoError(t, err)

	p.Content.Write(b)

	assert.Equal(t, b, p.Content.Bytes())
	assert.Equal(t, 1024, p.Content.Len())
	assert.GreaterOrEqual(t, p.Content.Cap(), 1024)

}

// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecycle(t *testing.T) {
	pool := NewPool()

	p := pool.Get()

	p.Metadata.Id = 32
	p.Metadata.Operation = 64
	p.Metadata.ContentLength = 128

	pool.Put(p)
	p = pool.Get()

	testData := make([]byte, p.Content.Cap()*2)
	_, err := rand.Read(testData)
	assert.NoError(t, err)
	for {
		assert.NotNil(t, p.Metadata)
		assert.Equal(t, uint16(0), p.Metadata.Id)
		assert.Equal(t, uint16(0), p.Metadata.Operation)
		assert.Equal(t, uint32(0), p.Metadata.ContentLength)
		assert.Equal(t, *pool.Get().Content, *p.Content)

		p.Content.Write(testData)
		assert.Equal(t, len(testData), p.Content.Len())
		assert.GreaterOrEqual(t, p.Content.Cap(), len(testData))

		pool.Put(p)
		p = pool.Get()

		assert.NotNil(t, p.Metadata)
		assert.Equal(t, uint16(0), p.Metadata.Id)
		assert.Equal(t, uint16(0), p.Metadata.Operation)
		assert.Equal(t, uint32(0), p.Metadata.ContentLength)

		if p.Content.Cap() < len(testData) {
			continue
		}
		assert.Equal(t, 0, p.Content.Len())
		assert.GreaterOrEqual(t, p.Content.Cap(), len(testData))
		break
	}

	pool.Put(p)
}

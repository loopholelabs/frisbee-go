/*
	Copyright 2022 Loophole Labs

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

		   http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package packet

import (
	"github.com/loopholelabs/polyglot-go"
	"math/rand"
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

	testData := make([]byte, cap(*p.Content)*2)
	_, err := rand.Read(testData)
	assert.NoError(t, err)
	for {
		assert.NotNil(t, p.Metadata)
		assert.Equal(t, uint16(0), p.Metadata.Id)
		assert.Equal(t, uint16(0), p.Metadata.Operation)
		assert.Equal(t, uint32(0), p.Metadata.ContentLength)
		assert.Equal(t, polyglot.Buffer{}, *p.Content)

		p.Content.Write(testData)
		assert.Equal(t, len(testData), len(*p.Content))
		assert.GreaterOrEqual(t, cap(*p.Content), len(testData))

		pool.Put(p)
		p = pool.Get()

		assert.NotNil(t, p.Metadata)
		assert.Equal(t, uint16(0), p.Metadata.Id)
		assert.Equal(t, uint16(0), p.Metadata.Operation)
		assert.Equal(t, uint32(0), p.Metadata.ContentLength)

		if cap(*p.Content) < len(testData) {
			continue
		}
		assert.Equal(t, 0, len(*p.Content))
		assert.GreaterOrEqual(t, cap(*p.Content), len(testData))
		break
	}

	pool.Put(p)
}

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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	p := Get()

	assert.IsType(t, new(Packet), p)
	assert.NotNil(t, p.Message)
	assert.Equal(t, uint16(0), p.Message.Id)
	assert.Equal(t, uint16(0), p.Message.Operation)
	assert.Equal(t, uint32(0), p.Message.ContentLength)
	assert.Nil(t, p.Content)

	Put(p)
}

func TestRecycle(t *testing.T) {
	t.Parallel()

	p := Get()

	p.Message.Id = 32
	p.Message.Operation = 64
	p.Message.ContentLength = 128

	Put(p)

	p = Get()
	assert.NotNil(t, p.Message)
	assert.Equal(t, uint16(0), p.Message.Id)
	assert.Equal(t, uint16(0), p.Message.Operation)
	assert.Equal(t, uint32(0), p.Message.ContentLength)
	assert.Nil(t, p.Content)

	p.Content = make([]byte, 32)
	assert.Equal(t, 32, len(p.Content))

	Put(p)
	p = Get()

	assert.NotNil(t, p.Message)
	assert.Equal(t, uint16(0), p.Message.Id)
	assert.Equal(t, uint16(0), p.Message.Operation)
	assert.Equal(t, uint32(0), p.Message.ContentLength)

	assert.NotNil(t, p.Content)
	assert.Equal(t, 0, len(p.Content))
	assert.Equal(t, 32, cap(p.Content))

	Put(p)
}

/*
	Copyright 2023 Loophole Labs

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

package polyglot

import (
	"sync"
)

var (
	pool = NewPool()
)

type Pool struct {
	pool sync.Pool
}

func NewPool() *Pool {
	return new(Pool)
}

func (p *Pool) Get() (b *Buffer) {
	v := p.pool.Get()
	if v == nil {
		b = NewBuffer()
		return
	}
	return v.(*Buffer)
}

func (p *Pool) Put(b *Buffer) {
	if b != nil {
		b.Reset()
		p.pool.Put(b)
	}
}

func GetBuffer() *Buffer {
	return pool.Get()
}

func PutBuffer(b *Buffer) {
	pool.Put(b)
}

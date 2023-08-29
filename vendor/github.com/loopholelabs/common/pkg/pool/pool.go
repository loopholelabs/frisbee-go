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

package pool

import "sync"

type PointerWithReset[T any] interface {
	*T

	Reset()
}

type Pool[T any, P PointerWithReset[T]] struct {
	pool sync.Pool
	New  func() P
}

func NewPool[T any, P PointerWithReset[T]](new func() P) *Pool[T, P] {
	return &Pool[T, P]{
		New: new,
	}
}

func (p *Pool[T, P]) Put(value P) {
	if value != nil {
		value.Reset()
		p.pool.Put(value)
	}
}

func (p *Pool[T, P]) Get() P {
	rv, ok := p.pool.Get().(P)
	if ok && rv != nil {
		return rv
	}

	return p.New()
}

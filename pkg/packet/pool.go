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
	"github.com/loopholelabs/frisbee/pkg/metadata"
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

func (p *Pool) Get() (s *Packet) {
	v := p.pool.Get()
	if v == nil {
		s = &Packet{
			Metadata: new(metadata.Metadata),
			Content:  make([]byte, 0, 512),
		}
		return
	}
	return v.(*Packet)
}

func (p *Pool) Put(packet *Packet) {
	if packet != nil {
		packet.Reset()
		p.pool.Put(packet)
	}
}

func Get() (s *Packet) {
	return pool.Get()
}

func Put(p *Packet) {
	pool.Put(p)
}

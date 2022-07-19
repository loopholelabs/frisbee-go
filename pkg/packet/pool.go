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
	"github.com/loopholelabs/common/pkg/pool"
	"github.com/loopholelabs/frisbee/pkg/metadata"
	"github.com/loopholelabs/polyglot-go"
)

var (
	packetPool = NewPool()
)

func NewPool() *pool.Pool[Packet, *Packet] {
	return pool.NewPool(func() *Packet {
		return &Packet{
			Metadata: new(metadata.Metadata),
			Content:  polyglot.NewBuffer(),
		}
	})
}

func Get() (s *Packet) {
	return packetPool.Get()
}

func Put(p *Packet) {
	packetPool.Put(p)
}

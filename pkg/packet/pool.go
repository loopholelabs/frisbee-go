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
	cpool "github.com/loopholelabs/common/pkg/pool"
	"github.com/loopholelabs/frisbee/pkg/content"
	"github.com/loopholelabs/frisbee/pkg/metadata"
)

var (
	pool = NewPool()
)

func NewPool() *cpool.Pool[Packet, *Packet] {
	return cpool.NewPool(func() *Packet {
		return &Packet{
			Metadata: new(metadata.Metadata),
			Content:  content.New(),
		}
	})
}

func Get() (s *Packet) {
	return pool.Get()
}

func Put(p *Packet) {
	pool.Put(p)
}

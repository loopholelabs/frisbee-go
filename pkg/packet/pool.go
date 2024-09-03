// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"github.com/loopholelabs/common/pkg/pool"
)

var (
	packetPool = NewPool()
)

func NewPool() *pool.Pool[Packet, *Packet] {
	return pool.NewPool(New)
}

func Get() (s *Packet) {
	return packetPool.Get()
}

func Put(p *Packet) {
	packetPool.Put(p)
}

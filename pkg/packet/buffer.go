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
	"github.com/pkg/errors"
	"sync"
)

var (
	BufferClosed = errors.New("packet buffer is closed")
)

type Buffer struct {
	p      []*Packet
	cond   *sync.Cond
	closed bool
}

func NewBuffer() *Buffer {
	return &Buffer{
		cond: sync.NewCond(new(sync.Mutex)),
	}
}

func (p *Buffer) Push(pushPacket *Packet) error {
	p.cond.L.Lock()
	if p.closed {
		p.cond.L.Unlock()
		return BufferClosed
	}
	p.p = append(p.p, pushPacket)
	p.cond.L.Unlock()
	p.cond.Signal()
	return nil
}

func (p *Buffer) Pop() (*Packet, error) {
	p.cond.L.Lock()
LOOP:
	if p.closed {
		p.cond.L.Unlock()
		return nil, BufferClosed
	}
	if len(p.p) == 0 {
		p.cond.Wait()
		goto LOOP
	}
	var popPacket *Packet
	popPacket, p.p = p.p[0], p.p[1:]
	p.cond.L.Unlock()
	return popPacket, nil
}

func (p *Buffer) Close() {
	p.cond.L.Lock()
	p.closed = true
	p.cond.L.Unlock()
	p.cond.Broadcast()
}

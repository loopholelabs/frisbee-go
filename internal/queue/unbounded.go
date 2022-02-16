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

package queue

import (
	"sync"
	"unsafe"
)

type Unbounded struct {
	data   []unsafe.Pointer
	cond   *sync.Cond
	closed bool
}

func NewUnbounded() *Unbounded {
	return &Unbounded{
		cond: sync.NewCond(new(sync.Mutex)),
	}
}

func (p *Unbounded) Push(item unsafe.Pointer) error {
	p.cond.L.Lock()
	if p.closed {
		p.cond.L.Unlock()
		return Closed
	}
	p.data = append(p.data, item)
	p.cond.L.Unlock()
	p.cond.Signal()
	return nil
}

func (p *Unbounded) Pop() (unsafe.Pointer, error) {
	p.cond.L.Lock()
LOOP:
	if p.closed {
		p.cond.L.Unlock()
		return nil, Closed
	}
	if len(p.data) == 0 {
		p.cond.Wait()
		goto LOOP
	}
	var item unsafe.Pointer
	item, p.data = p.data[0], p.data[1:]
	p.cond.L.Unlock()
	return item, nil
}

func (p *Unbounded) Close() {
	p.cond.L.Lock()
	p.closed = true
	p.cond.L.Unlock()
	p.cond.Broadcast()
}

func (p *Unbounded) IsClosed() (b bool) {
	p.cond.L.Lock()
	b = p.closed
	p.cond.L.Unlock()
	return
}

func (p *Unbounded) Length() (length int) {
	p.cond.L.Lock()
	length = len(p.data)
	p.cond.L.Unlock()
	return
}

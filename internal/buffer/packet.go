package buffer

import (
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/pkg/errors"
	"sync"
)

var (
	PacketBufferClosed = errors.New("packet buffer is closed")
)

type Packet struct {
	p      []*packet.Packet
	mu     sync.Mutex
	cond   *sync.Cond
	closed bool
}

func NewPacket(size int) (p *Packet) {
	p = &Packet{
		p: make([]*packet.Packet, 0, size),
	}

	p.cond = sync.NewCond(&p.mu)
	return
}

func (p *Packet) Push(pushPacket *packet.Packet) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return PacketBufferClosed
	}
	p.p = append(p.p, pushPacket)
	p.mu.Unlock()
	p.cond.Signal()
	return nil
}

func (p *Packet) Pop() (*packet.Packet, error) {
	p.mu.Lock()
LOOP:
	if p.closed {
		p.mu.Unlock()
		return nil, PacketBufferClosed
	}
	if len(p.p) == 0 {
		p.cond.Wait()
		goto LOOP
	}
	var popPacket *packet.Packet
	popPacket, p.p = p.p[0], p.p[1:]
	p.mu.Unlock()
	return popPacket, nil
}

func (p *Packet) Close() {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	p.cond.Broadcast()
}

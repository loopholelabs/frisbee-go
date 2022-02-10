package metadata

import "sync"

type Buffer [Size]byte

type Pool struct {
	pool sync.Pool
}

func NewPool() *Pool {
	return new(Pool)
}

func (p *Pool) Get() Buffer {
	v := p.pool.Get()
	if v == nil {
		v = Buffer{}
	}
	return v.(Buffer)
}

func (p *Pool) Put(b Buffer) {
	p.pool.Put(b)
}

var (
	pool = NewPool()
)

func Get() Buffer {
	return pool.Get()
}

func Put(b Buffer) {
	pool.Put(b)
}

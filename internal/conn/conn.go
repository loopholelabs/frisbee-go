package conn

import (
	"github.com/panjf2000/gnet"
	"net"
)

type Conn struct {
	conn gnet.Conn
}

func New(c interface{}) Conn {
	return Conn{
		conn: c.(gnet.Conn),
	}
}

func (c *Conn) Context() interface{} {
	return c.conn.Context()
}

func (c *Conn) SetContext(ctx interface{}) {
	c.conn.SetContext(ctx)
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) Read() []byte {
	return c.conn.Read()
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

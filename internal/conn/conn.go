package conn

import (
	"github.com/panjf2000/gnet"
	"github.com/pkg/errors"
	"net"
	"sync"
)

type Conn string

type internalConn struct {
	conn gnet.Conn
	mu   sync.RWMutex
}

type Connections struct {
	connectedSockets sync.Map
}

var connections Connections

func Convert(c interface{}) (conn Conn) {
	if c == nil {
		panic(errors.New("interface was nil"))
	}

	storedConnection := &internalConn{
		conn: c.(gnet.Conn),
	}
	conn = Conn(storedConnection.conn.RemoteAddr().String())
	connections.connectedSockets.Store(conn, storedConnection)
	return
}

func (c Conn) Context() interface{} {
	connection, _ := connections.connectedSockets.Load(c)
	if connection != nil {
		(*connection.(*internalConn)).mu.RLock()
		defer (*connection.(*internalConn)).mu.RUnlock()
		return (*connection.(*internalConn)).conn.(gnet.Conn).Context()
	}
	return errors.New("invalid connection")
}

func (c Conn) SetContext(ctx interface{}) {
	connection, _ := connections.connectedSockets.Load(c)
	if connection != nil {
		(*connection.(*internalConn)).mu.Lock()
		defer (*connection.(*internalConn)).mu.Unlock()
		(*connection.(*internalConn)).conn.(gnet.Conn).SetContext(ctx)
	}
}

func (c Conn) LocalAddr() net.Addr {
	connection, _ := connections.connectedSockets.Load(c)
	if connection != nil {
		(*connection.(*internalConn)).mu.RLock()
		defer (*connection.(*internalConn)).mu.RUnlock()
		return (*connection.(*internalConn)).conn.(gnet.Conn).LocalAddr()
	}
	return nil
}

func (c Conn) RemoteAddr() net.Addr {
	connection, _ := connections.connectedSockets.Load(c)
	if connection != nil {
		(*connection.(*internalConn)).mu.RLock()
		defer (*connection.(*internalConn)).mu.RUnlock()
		return (*connection.(*internalConn)).conn.(gnet.Conn).RemoteAddr()
	}
	return nil
}

func (c Conn) AsyncWrite(data []byte) error {
	connection, _ := connections.connectedSockets.Load(c)
	if connection != nil {
		(*connection.(*internalConn)).mu.Lock()
		defer (*connection.(*internalConn)).mu.Unlock()
		return (*connection.(*internalConn)).conn.(gnet.Conn).AsyncWrite(data)
	}
	return errors.New("invalid connection")
}

func (c Conn) Close() error {
	connection, _ := connections.connectedSockets.Load(c)
	if connection != nil {
		(*connection.(*internalConn)).mu.Lock()
		defer (*connection.(*internalConn)).mu.Unlock()
		connections.connectedSockets.Delete(c)
		return (*connection.(*internalConn)).conn.(gnet.Conn).Close()
	}
	return errors.New("invalid connection")
}

package server

import (
	"github.com/loophole-labs/frisbee"
	"net"
	"time"
)

type Server struct {
	listener           *net.TCPListener
	addr               string
	router             frisbee.ServerRouter
	Options            *Options
	UserOnInitComplete func(server *Server) frisbee.Action
	UserOnOpened       func(server *Server, c frisbee.Conn) frisbee.Action
	UserOnClosed       func(server *Server, c frisbee.Conn, err error) frisbee.Action
	UserOnShutdown     func(server *Server)
	UserPreWrite       func(server *Server)
	UserTick           func(server *Server) (time.Duration, frisbee.Action)
}

func NewServer() {

}

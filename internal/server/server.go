package server

import (
	"github.com/loophole-labs/frisbee/internal/common"
	"github.com/loophole-labs/frisbee/internal/conn"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"net"
)

type RouterFunc func(c *conn.Conn, incomingMessage protocol.MessageV0, incomingContent []byte) (outgoingMessage *protocol.MessageV0, outgoingContent []byte, action int)
type Router map[uint16]RouterFunc

type Server struct {
	listener       *net.TCPListener
	addr           string
	router         Router
	shutdown       bool
	Options        *Options
	UserOnOpened   func(server *Server, c *conn.Conn) int
	UserOnClosed   func(server *Server, c *conn.Conn, err error) int
	UserOnShutdown func(server *Server)
	UserPreWrite   func(server *Server)
}

func NewServer(addr string, router Router, opts ...Option) *Server {
	return &Server{
		addr:    addr,
		router:  router,
		Options: LoadOptions(opts...),
	}
}

func (s *Server) onOpened(c *conn.Conn) int {
	return s.UserOnOpened(s, c)
}

func (s *Server) onClosed(c *conn.Conn, err error) int {
	return s.UserOnClosed(s, c, err)
}

func (s *Server) onShutdown() {
	s.UserOnShutdown(s)
}

func (s *Server) preWrite() {
	s.UserPreWrite(s)
}

func (s *Server) Start() error {

	if s.UserOnClosed == nil {
		s.UserOnClosed = func(_ *Server, _ *conn.Conn, err error) int {
			return common.None
		}
	}

	if s.UserOnOpened == nil {
		s.UserOnOpened = func(_ *Server, _ *conn.Conn) int {
			return common.None
		}
	}

	if s.UserOnShutdown == nil {
		s.UserOnShutdown = func(_ *Server) {}
	}

	if s.UserPreWrite == nil {
		s.UserPreWrite = func(_ *Server) {}
	}

	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = l.(*net.TCPListener)

	go func() {
		for {
			newConn, err := l.Accept()
			if err != nil {
				if s.shutdown {
					return
				}
				s.Options.Logger.Fatal().Msgf("unable to accept connections: %+v", err)
				return
			}
			go s.handleConn(newConn)
		}
	}()

	return nil
}

func (s *Server) handleConn(newConn net.Conn) {
	_ = newConn.(*net.TCPConn).SetKeepAlive(true)
	_ = newConn.(*net.TCPConn).SetKeepAlivePeriod(s.Options.KeepAlive)
	frisbeeConn := conn.New(newConn)

	openedAction := s.UserOnOpened(s, frisbeeConn)
	if openedAction == common.Close {
		_ = frisbeeConn.Close()
		s.onClosed(frisbeeConn, nil)
		return
	}

	if openedAction == common.Shutdown {
		_ = frisbeeConn.Close()
		s.onClosed(frisbeeConn, nil)
		_ = s.Shutdown()
		s.onShutdown()
		return
	}

	for {
		incomingMessage, incomingContent, err := frisbeeConn.Read()
		if err != nil {
			_ = frisbeeConn.Close()
			s.onClosed(frisbeeConn, err)
			return
		}
		routerFunc := s.router[incomingMessage.Operation]
		if routerFunc != nil {
			outgoingMessage, outgoingContent, action := routerFunc(frisbeeConn, *incomingMessage, *incomingContent)

			if outgoingMessage != nil && outgoingMessage.ContentLength == uint32(len(outgoingContent)) {
				s.preWrite()
				err := frisbeeConn.Write(outgoingMessage, &outgoingContent)
				if err != nil {
					_ = frisbeeConn.Close()
					s.onClosed(frisbeeConn, err)
					return
				}
			}

			switch action {
			case common.Close:
				_ = frisbeeConn.Close()
				s.onClosed(frisbeeConn, nil)
				return
			case common.Shutdown:
				_ = frisbeeConn.Close()
				s.UserOnClosed(s, frisbeeConn, nil)
				_ = s.Shutdown()
				s.UserOnShutdown(s)
				return
			default:
			}
		}
	}
}

func (s *Server) Shutdown() error {
	s.shutdown = true
	return s.listener.Close()
}

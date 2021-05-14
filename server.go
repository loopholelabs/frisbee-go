package frisbee

import (
	"github.com/loophole-labs/frisbee/internal/errors"
	"github.com/rs/zerolog"
	"net"
)

type ServerRouterFunc func(c *Conn, incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)
type ServerRouter map[uint32]ServerRouterFunc

type Server struct {
	listener   *net.TCPListener
	addr       string
	router     ServerRouter
	shutdown   bool
	Options    *Options
	OnOpened   func(server *Server, c *Conn) Action
	OnClosed   func(server *Server, c *Conn, err error) Action
	OnShutdown func(server *Server)
	PreWrite   func(server *Server)
}

func NewServer(addr string, router ServerRouter, opts ...Option) *Server {
	return &Server{
		addr:    addr,
		router:  router,
		Options: loadOptions(opts...),
	}
}

func (s *Server) onOpened(c *Conn) Action {
	return s.OnOpened(s, c)
}

func (s *Server) onClosed(c *Conn, err error) Action {
	return s.OnClosed(s, c, err)
}

func (s *Server) onShutdown() {
	s.OnShutdown(s)
}

func (s *Server) preWrite() {
	s.PreWrite(s)
}

func (s *Server) Start() error {

	if s.OnClosed == nil {
		s.OnClosed = func(_ *Server, _ *Conn, err error) Action {
			return None
		}
	}

	if s.OnOpened == nil {
		s.OnOpened = func(_ *Server, _ *Conn) Action {
			return None
		}
	}

	if s.OnShutdown == nil {
		s.OnShutdown = func(_ *Server) {}
	}

	if s.PreWrite == nil {
		s.PreWrite = func(_ *Server) {}
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
				s.logger().Fatal().Msgf(errors.WithContext(err, ACCEPT).Error())
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
	frisbeeConn := New(newConn, nil)

	openedAction := s.onOpened(frisbeeConn)

	switch openedAction {
	case Close:
		_ = frisbeeConn.Close()
		s.onClosed(frisbeeConn, nil)
		return
	case Shutdown:
		_ = frisbeeConn.Close()
		s.onClosed(frisbeeConn, nil)
		_ = s.Shutdown()
		s.onShutdown()
		return
	default:
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
			var outgoingMessage *Message
			var outgoingContent []byte
			var action Action
			if incomingMessage.ContentLength == 0 || incomingContent == nil {
				outgoingMessage, outgoingContent, action = routerFunc(frisbeeConn, *incomingMessage, nil)
			} else {
				outgoingMessage, outgoingContent, action = routerFunc(frisbeeConn, *incomingMessage, *incomingContent)
			}

			if outgoingMessage != nil && outgoingMessage.ContentLength == uint64(len(outgoingContent)) {
				s.preWrite()
				err := frisbeeConn.Write(outgoingMessage, &outgoingContent)
				if err != nil {
					_ = frisbeeConn.Close()
					s.onClosed(frisbeeConn, err)
					return
				}
			}

			switch action {
			case Close:
				_ = frisbeeConn.Close()
				s.onClosed(frisbeeConn, nil)
				return
			case Shutdown:
				_ = frisbeeConn.Close()
				s.OnClosed(s, frisbeeConn, nil)
				_ = s.Shutdown()
				s.OnShutdown(s)
				return
			default:
			}
		}
	}
}

func (s *Server) logger() *zerolog.Logger {
	return s.Options.Logger
}

func (s *Server) Shutdown() error {
	s.shutdown = true
	return s.listener.Close()
}

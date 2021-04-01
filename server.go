package frisbee

import (
	"net"
)

type ServerRouterFunc func(c *Conn, incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)
type ServerRouter map[uint16]ServerRouterFunc

type Server struct {
	listener       *net.TCPListener
	addr           string
	router         ServerRouter
	shutdown       bool
	Options        *Options
	UserOnOpened   func(server *Server, c *Conn) Action
	UserOnClosed   func(server *Server, c *Conn, err error) Action
	UserOnShutdown func(server *Server)
	UserPreWrite   func(server *Server)
}

func NewServer(addr string, router ServerRouter, opts ...Option) *Server {
	return &Server{
		addr:    addr,
		router:  router,
		Options: LoadOptions(opts...),
	}
}

func (s *Server) onOpened(c *Conn) Action {
	return s.UserOnOpened(s, c)
}

func (s *Server) onClosed(c *Conn, err error) Action {
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
		s.UserOnClosed = func(_ *Server, _ *Conn, err error) Action {
			return None
		}
	}

	if s.UserOnOpened == nil {
		s.UserOnOpened = func(_ *Server, _ *Conn) Action {
			return None
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
	frisbeeConn := New(newConn)

	openedAction := s.onOpened(frisbeeConn)
	if openedAction == Close {
		_ = frisbeeConn.Close()
		s.onClosed(frisbeeConn, nil)
		return
	}

	if openedAction == Shutdown {
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
			var outgoingMessage *Message
			var outgoingContent []byte
			var action Action
			if incomingMessage.ContentLength == 0 || incomingContent == nil {
				outgoingMessage, outgoingContent, action = routerFunc(frisbeeConn, Message(*incomingMessage), nil)
			} else {
				outgoingMessage, outgoingContent, action = routerFunc(frisbeeConn, Message(*incomingMessage), *incomingContent)
			}

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
			case Close:
				_ = frisbeeConn.Close()
				s.onClosed(frisbeeConn, nil)
				return
			case Shutdown:
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

package frisbee

import (
	"github.com/loophole-labs/frisbee/internal/errors"
	"github.com/rs/zerolog"
	"net"
	"time"
)

// ServerRouterFunc defines a message handler for a specific frisbee message
type ServerRouterFunc func(c *Conn, incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action)

// ServerRouter maps frisbee message types to specific handler functions (of type ServerRouterFunc)
type ServerRouter map[uint32]ServerRouterFunc

// Server accepts connections from frisbee Clients and can send and receive frisbee messages
type Server struct {
	listener      *net.TCPListener
	addr          string
	router        ServerRouter
	shutdown      bool
	options       *Options
	messageOffset uint32
	OnOpened      func(server *Server, c *Conn) Action
	OnClosed      func(server *Server, c *Conn, err error) Action
	OnShutdown    func(server *Server)
	PreWrite      func(server *Server)
}

// NewServer returns an uninitialized frisbee Server with the registered ServerRouter.
// The Start method must then be called to start the server and listen for connections
func NewServer(addr string, router ServerRouter, opts ...Option) *Server {

	options := loadOptions(opts...)
	messageOffset := uint32(0)
	newRouter := ServerRouter{}

	if options.Heartbeat != time.Duration(-1) {
		newRouter[messageOffset] = func(c *Conn, incomingMessage Message, incomingContent []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
			outgoingMessage = &Message{
				From:          incomingMessage.From,
				To:            incomingMessage.To,
				Id:            incomingMessage.Id,
				Operation:     HEARTBEAT - c.Offset(),
				ContentLength: incomingMessage.ContentLength,
			}
			return
		}

		messageOffset++
	}

	for message, handler := range router {
		newRouter[message+messageOffset] = handler
	}

	return &Server{
		addr:          addr,
		router:        newRouter,
		options:       options,
		messageOffset: messageOffset,
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
			return NONE
		}
	}

	if s.OnOpened == nil {
		s.OnOpened = func(_ *Server, _ *Conn) Action {
			return NONE
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
				s.Logger().Fatal().Msgf(errors.WithContext(err, ACCEPT).Error())
				return
			}
			go s.handleConn(newConn)
		}
	}()

	return nil
}

func (s *Server) handleConn(newConn net.Conn) {
	_ = newConn.(*net.TCPConn).SetKeepAlive(true)
	_ = newConn.(*net.TCPConn).SetKeepAlivePeriod(s.options.KeepAlive)
	frisbeeConn := New(newConn, s.Logger(), s.messageOffset)

	openedAction := s.onOpened(frisbeeConn)

	switch openedAction {
	case CLOSE:
		_ = frisbeeConn.Close()
		s.onClosed(frisbeeConn, nil)
		return
	case SHUTDOWN:
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
			case CLOSE:
				_ = frisbeeConn.Close()
				s.onClosed(frisbeeConn, nil)
				return
			case SHUTDOWN:
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

func (s *Server) Logger() *zerolog.Logger {
	return s.options.Logger
}

func (s *Server) Shutdown() error {
	s.shutdown = true
	return s.listener.Close()
}

package server

import (
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/handler"
	"time"
)

type Server struct {
	*handler.Handler
	addr               string
	router             frisbee.ServerRouter
	Custom             interface{}
	Options            *Options
	UserOnInitComplete func(server *Server) frisbee.Action
	UserOnOpened       func(server *Server, c frisbee.Conn) ([]byte, frisbee.Action)
	UserOnClosed       func(server *Server, c frisbee.Conn, err error) frisbee.Action
	UserOnShutdown     func(server *Server)
	UserPreWrite       func(server *Server)
	UserTick           func(server *Server) (time.Duration, frisbee.Action)
}

func NewServer(addr string, router frisbee.ServerRouter, opts ...Option) *Server {
	return &Server{
		addr:    addr,
		router:  router,
		Options: LoadOptions(opts...),
	}
}

func (s *Server) onInitComplete() frisbee.Action {
	return s.UserOnInitComplete(s)
}

func (s *Server) onOpened(c frisbee.Conn) ([]byte, frisbee.Action) {
	return s.UserOnOpened(s, c)
}

func (s *Server) onClosed(c frisbee.Conn, err error) frisbee.Action {
	return s.UserOnClosed(s, c, err)
}

func (s *Server) onShutdown() {
	s.UserOnShutdown(s)
}

func (s *Server) preWrite() {
	s.UserPreWrite(s)
}

func (s *Server) tick() (time.Duration, frisbee.Action) {
	return s.UserTick(s)
}

func (s *Server) Start() {

	if s.UserOnClosed == nil {
		s.UserOnClosed = func(_ *Server, _ frisbee.Conn, err error) frisbee.Action {
			return frisbee.None
		}
	}

	if s.UserOnOpened == nil {
		s.UserOnOpened = func(_ *Server, _ frisbee.Conn) ([]byte, frisbee.Action) {
			return nil, frisbee.None
		}
	}

	if s.UserOnInitComplete == nil {
		s.UserOnInitComplete = func(_ *Server) frisbee.Action {
			return frisbee.None
		}
	}

	if s.UserOnShutdown == nil {
		s.UserOnShutdown = func(_ *Server) {}
	}

	if s.UserPreWrite == nil {
		s.UserPreWrite = func(_ *Server) {}
	}

	if s.UserTick == nil {
		s.UserTick = func(_ *Server) (delay time.Duration, action frisbee.Action) {
			return
		}
	}

	started := make(chan struct{})
	s.Options.Logger.Info().Msg("Starting Server")
	s.Handler = handler.StartHandler(
		started,
		s.addr,
		s.Options.Multicore,
		s.Options.Async,
		s.Options.Loops,
		s.Options.KeepAlive,
		s.Options.Logger,
		s.router,
		s.onInitComplete,
		s.onOpened,
		s.onClosed,
		s.onShutdown,
		s.preWrite,
		s.tick)
	<-started
	return
}

func (s *Server) Stop() error {
	s.Options.Logger.Info().Msg("Stopping Server")
	return s.Handler.Stop()
}

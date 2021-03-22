package server

import (
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/handler"
)

type Server struct {
	*handler.Handler
	addr    string
	router  frisbee.Router
	options *Options
}

func NewServer(addr string, router frisbee.Router, opts ...Option) *Server {
	return &Server{
		addr:    addr,
		router:  router,
		options: LoadOptions(opts...),
	}
}

func (s *Server) Start() {
	started := make(chan struct{})
	s.options.Logger.Info().Msg("Starting Server")
	s.Handler = handler.StartHandler(
		started,
		s.addr,
		s.options.Multicore,
		s.options.Async,
		s.options.Loops,
		s.options.KeepAlive,
		s.options.Logger,
		s.router)
	<-started
	return
}

func (s *Server) Stop() error {
	s.options.Logger.Info().Msg("Stopping Server")
	return s.Handler.Stop()
}

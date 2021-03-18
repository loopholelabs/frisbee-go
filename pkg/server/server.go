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
		options: loadOptions(opts...),
	}
}

func (s *Server) Start() error {
	started := make(chan struct{})
	serverError := make(chan error)
	s.options.logger.Info().Msg("Starting Server")
	s.Handler = handler.StartHandler(
		started,
		serverError,
		s.addr, s.options.multicore,
		s.options.async,
		s.options.loops,
		s.options.keepAlive,
		s.options.logger,
		s.router)
	<-started
	err := <-serverError
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Stop() error {
	s.options.logger.Info().Msg("Stopping Server")
	return s.Handler.Stop()
}

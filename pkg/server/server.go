package server

import (
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/handler"
)

type Server struct {
	*handler.Handler
	addr    string
	router  frisbee.Router
	options *frisbee.Options
}

func NewServer(addr string, router frisbee.Router, opts ...frisbee.Option) *Server {
	return &Server{
		addr:    addr,
		router:  router,
		options: frisbee.LoadOptions(opts...),
	}
}

func (s *Server) Start() error {
	started := make(chan struct{})
	serverError := make(chan error)
	s.options.Logger.Info().Msg("Starting Server")
	s.Handler = handler.StartHandler(
		started,
		serverError,
		s.addr, s.options.Multicore,
		s.options.Async,
		s.options.Loops,
		s.options.KeepAlive,
		s.options.Logger,
		s.router)
	<-started
	err := <-serverError
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Stop() error {
	s.options.Logger.Info().Msg("Stopping Server")
	return s.Handler.Stop()
}

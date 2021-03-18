package server

import (
	"github.com/loophole-labs/frisbee/internal/handler"
)

type Router handler.Router

type Server struct {
	*handler.Handler
	addr    string
	router  Router
	options *Options
}

func NewServer(addr string, router Router, opts ...Option) *Server {
	return &Server{
		addr:    addr,
		router:  router,
		options: loadOptions(opts...),
	}
}

func (s *Server) Start() {
	started := make(chan struct{})
	s.Handler = handler.StartHandler(
		started,
		s.addr, s.options.multicore,
		s.options.async,
		s.options.loops,
		s.options.keepAlive,
		s.options.logger,
		handler.Router(s.router))
	<-started
}

package server

import (
	"github.com/rs/zerolog"
	"os"
	"time"
)

type Option func(opts *Options)

var DefaultLogger = zerolog.New(os.Stdout)

type Options struct {
	multicore bool
	async     bool
	loops     int
	keepAlive time.Duration
	logger    *zerolog.Logger
}

func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}

	if opts.loops < -1 {
		opts.loops = -1
	}

	if opts.logger == nil {
		opts.logger = &DefaultLogger
	}

	return opts
}

func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

func WithMulticore(multicore bool) Option {
	return func(opts *Options) {
		opts.multicore = multicore
	}
}

func WithAsync(async bool) Option {
	return func(opts *Options) {
		opts.async = async
	}
}

func WithLoops(loops int) Option {
	return func(opts *Options) {
		opts.loops = loops
	}
}

func WithKeepAlive(keepAlive time.Duration) Option {
	return func(opts *Options) {
		opts.keepAlive = keepAlive
	}
}

func WithLogger(logger *zerolog.Logger) Option {
	return func(opts *Options) {
		opts.logger = logger
	}
}

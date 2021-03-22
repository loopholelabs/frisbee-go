package server

import (
	"github.com/rs/zerolog"
	"os"
	"time"
)

type Option func(opts *Options)

var DefaultLogger = zerolog.New(os.Stdout)

type Options struct {
	Multicore bool
	Async     bool
	Loops     int
	KeepAlive time.Duration
	Logger    *zerolog.Logger
}

func LoadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}

	if opts.Loops <= 0 {
		opts.Loops = 16
	}

	if opts.Logger == nil {
		opts.Logger = &DefaultLogger
	}

	if opts.KeepAlive == 0 {
		opts.KeepAlive = time.Minute * 3
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
		opts.Multicore = multicore
	}
}

func WithAsync(async bool) Option {
	return func(opts *Options) {
		opts.Async = async
	}
}

func WithLoops(loops int) Option {
	return func(opts *Options) {
		opts.Loops = loops
	}
}

func WithKeepAlive(keepAlive time.Duration) Option {
	return func(opts *Options) {
		opts.KeepAlive = keepAlive
	}
}

func WithLogger(logger *zerolog.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

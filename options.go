package frisbee

import (
	"github.com/rs/zerolog"
	"os"
	"time"
)

// Option is used to generate frisbee client and server options internally
type Option func(opts *Options)

// DefaultLogger is the default logger used within frisbee
var DefaultLogger = zerolog.New(os.Stdout)

// Options is used to provide the frisbee client and server with configuration options.
//
// Default Values:
//	options := Options {
//		KeepAlive: time.Minute * 3,
//		Logger: &DefaultLogger,
//		Heartbeat: time.Second * 5,
//	}
type Options struct {
	KeepAlive time.Duration
	Heartbeat time.Duration
	Logger    *zerolog.Logger
}

func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}

	if opts.Logger == nil {
		opts.Logger = &DefaultLogger
	}

	if opts.KeepAlive == 0 {
		opts.KeepAlive = time.Minute * 3
	}

	if opts.Heartbeat == 0 {
		opts.Heartbeat = time.Second * 5
	}

	return opts
}

// WithOptions allows users to pass in an Options struct to configure a frisbee client or server
func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

// WithKeepAlive allows users to define TCP keepalive options for the frisbee client or server (use -1 to disable)
func WithKeepAlive(keepAlive time.Duration) Option {
	return func(opts *Options) {
		opts.KeepAlive = keepAlive
	}
}

// WithLogger sets the logger for the frisbee client or server
func WithLogger(logger *zerolog.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

// WithHeartbeat sets the minimum time between heartbeat messages. By default, messages are only sent if
// no messages have been sent since the last heartbeat message - to change this behaviour you can disable heartbeats
// (by passing in -1), and implementing your own logic.
func WithHeartbeat(heartbeat time.Duration) Option {
	return func(opts *Options) {
		opts.Heartbeat = heartbeat
	}
}

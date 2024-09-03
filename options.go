// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"crypto/tls"
	"time"

	"github.com/loopholelabs/logging/loggers/noop"
	"github.com/loopholelabs/logging/types"
)

// Option is used to generate frisbee client and server options internally
type Option func(opts *Options)

// Options is used to provide the frisbee client and server with configuration options.
//
// Default Values:
//
//	options := Options {
//		KeepAlive: time.Minute * 3,
//		Logger: &DefaultLogger,
//	}
type Options struct {
	KeepAlive time.Duration
	Logger    types.Logger
	TLSConfig *tls.Config
}

func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}

	if opts.Logger == nil {
		opts.Logger = noop.New(types.InfoLevel)
	} else {
		opts.Logger = opts.Logger.SubLogger("frisbee")
	}

	if opts.KeepAlive == 0 {
		opts.KeepAlive = time.Minute * 3
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
func WithLogger(logger types.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

// WithTLS sets the TLS configuration for Frisbee. By default, no TLS configuration is used, and
// Frisbee will use unencrypted TCP connections. If the Frisbee Server is using TLS, then you must pass in
// a TLS config (even an empty one `&tls.Config{}`) for the Frisbee Client.
func WithTLS(tlsConfig *tls.Config) Option {
	return func(opts *Options) {
		opts.TLSConfig = tlsConfig
	}
}

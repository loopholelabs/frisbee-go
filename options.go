/*
	Copyright 2021 Loophole Labs

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

		   http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package frisbee

import (
	"crypto/tls"
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
	TLSConfig *tls.Config
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

// WithHeartbeat sets the minimum time between heartbeat packets. By default, packets are only sent if
// no packets have been sent since the last heartbeat packet - to change this behaviour you can disable heartbeats
// (by passing in -1), and implementing your own logic.
func WithHeartbeat(heartbeat time.Duration) Option {
	return func(opts *Options) {
		opts.Heartbeat = heartbeat
	}
}

// WithTLS sets the TLS configuration for Frisbee. By default no TLS configuration is used, and
// Frisbee will use unencrypted TCP connections. If the Frisbee Server is using TLS, then you must pass in
// a TLS config (even an empty one `&tls.Config{}`) for the Frisbee Client.
func WithTLS(tlsConfig *tls.Config) Option {
	return func(opts *Options) {
		opts.TLSConfig = tlsConfig
	}
}

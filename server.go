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
	"context"
	"crypto/tls"
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"net"
	"sync"
	"time"
)

var (
	defaultBaseContext = func() context.Context {
		return context.Background()
	}

	defaultOnClosed = func(_ *Async, _ error) {}

	defaultPreWrite = func() {}
)

// Server accepts connections from frisbee Clients and can send and receive frisbee messages
type Server struct {
	listener     net.Listener
	addr         string
	handlerTable HandlerTable
	shutdown     *atomic.Bool
	options      *Options
	wg           sync.WaitGroup
	pool         *ants.Pool

	// BaseContext is used to define the base context for this Server and all incoming connections
	BaseContext func() context.Context

	// ConnContext is used to define a connection-specific context based on the incoming connection
	// and is run whenever a new connection is opened
	ConnContext func(context.Context, *Async) context.Context

	// PacketContext is used to define a packet-specific context based on the incoming packet
	// for a connection, and is run every time a new packet is being handled
	PacketContext func(context.Context, *packet.Packet) context.Context

	// OnClosed is a function run by the server whenever a connection is closed
	OnClosed func(*Async, error)

	// PreWrite is run by the server before a write happens
	PreWrite func()
}

// NewServer returns an uninitialized frisbee Server with the registered HandlerTable.
// The Start method must then be called to start the server and listen for connections
func NewServer(addr string, handlerTable HandlerTable, poolSize int, opts ...Option) (*Server, error) {
	for i := uint16(0); i < RESERVED9; i++ {
		if _, ok := handlerTable[i]; ok {
			return nil, InvalidHandlerTable
		}
	}

	options := loadOptions(opts...)
	if options.Heartbeat > time.Duration(0) {
		handlerTable[HEARTBEAT] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
			outgoing = incoming
			return
		}
	}

	if poolSize < 100 {
		poolSize = 100
	}

	pool, err := ants.NewPool(poolSize, ants.WithPreAlloc(true))
	if err != nil {
		return nil, err
	}

	return &Server{
		addr:         addr,
		handlerTable: handlerTable,
		options:      options,
		shutdown:     atomic.NewBool(false),
		pool:         pool,
	}, nil
}

// Start will start the frisbee server and its reactor goroutines
// to receive and handle incoming connections. If the BaseContext, ConnContext,
// OnClosed, OnShutdown, or PreWrite functions have not been defined, it will
// use the default functions for these.
func (s *Server) Start() error {

	if s.BaseContext == nil {
		s.BaseContext = defaultBaseContext
	}

	if s.OnClosed == nil {
		s.OnClosed = defaultOnClosed
	}

	if s.PreWrite == nil {
		s.PreWrite = defaultPreWrite
	}

	var err error
	if s.options.TLSConfig != nil {
		s.listener, err = tls.Listen("tcp", s.addr, s.options.TLSConfig)
	} else {
		s.listener, err = net.Listen("tcp", s.addr)
	}

	if err != nil {
		return err
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			newConn, err := s.listener.Accept()
			if err != nil {
				if s.shutdown.Load() {
					return
				}
				s.Logger().Fatal().Err(err).Msg("error while accepting connection")
				return
			}
			go s.handleConn(newConn)
		}
	}()

	return nil
}

func (s *Server) handlePacket(ctx context.Context, p *packet.Packet, conn *Async) func() {
	return func() {
		handlerFunc := s.handlerTable[p.Metadata.Operation]
		if handlerFunc != nil {
			outgoing, action := handlerFunc(ctx, p)
			if outgoing != nil && outgoing.Metadata.ContentLength == uint32(len(outgoing.Content)) {
				s.PreWrite()
				err := conn.WritePacket(outgoing)
				if err != nil {
					_ = conn.Close()
					s.OnClosed(conn, err)
					return
				}
			}
			if outgoing != p {
				packet.Put(outgoing)
			}
			packet.Put(p)
			switch action {
			case NONE:
			case CLOSE:
				_ = conn.Close()
				s.OnClosed(conn, nil)
			case SHUTDOWN:
				_ = conn.Close()
				s.OnClosed(conn, nil)
				_ = s.Shutdown()
			}
		}
	}
}

func (s *Server) handleConn(newConn net.Conn) {
	switch v := newConn.(type) {
	case *net.TCPConn:
		_ = v.SetKeepAlive(true)
		_ = v.SetKeepAlivePeriod(s.options.KeepAlive)
	}

	frisbeeConn := NewAsync(newConn, s.Logger())
	connCtx := s.BaseContext()

	if s.ConnContext != nil {
		connCtx = s.ConnContext(connCtx, frisbeeConn)
	}

	var p *packet.Packet
	var err error
	for {
		p, err = frisbeeConn.ReadPacket()
		if err != nil {
			_ = frisbeeConn.Close()
			s.OnClosed(frisbeeConn, err)
			return
		}
		packetCtx := connCtx
		if s.PacketContext != nil {
			packetCtx = s.PacketContext(packetCtx, p)
		}
		err = s.pool.Submit(s.handlePacket(packetCtx, p, frisbeeConn))
		if err != nil {
			_ = frisbeeConn.Close()
			s.OnClosed(frisbeeConn, err)
			return
		}
	}
}

// Logger returns the server's logger (useful for ServerRouter functions)
func (s *Server) Logger() *zerolog.Logger {
	return s.options.Logger
}

// Shutdown shuts down the frisbee server and kills all the goroutines and active connections
func (s *Server) Shutdown() error {
	s.shutdown.Store(true)
	s.pool.Release()
	defer s.wg.Wait()
	return s.listener.Close()
}

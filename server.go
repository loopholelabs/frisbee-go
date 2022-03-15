/*
	Copyright 2022 Loophole Labs

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
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"net"
	"sync"
)

var (
	defaultBaseContext = func() context.Context {
		return context.Background()
	}

	defaultOnClosed = func(_ *Async, _ error) {}

	defaultPreWrite = func() {}
)

// Server accepts connections from frisbee Clients and can send and receive frisbee Packets
type Server struct {
	listener      net.Listener
	addr          string
	handlerTable  HandlerTable
	shutdown      *atomic.Bool
	options       *Options
	wg            sync.WaitGroup
	connections   map[*Async]struct{}
	connectionsMu sync.Mutex

	// BaseContext is used to define the base context for this Server and all incoming connections
	BaseContext func() context.Context

	// ConnContext is used to define a connection-specific context based on the incoming connection
	// and is run whenever a new connection is opened
	ConnContext func(context.Context, *Async) context.Context

	// PacketContext is used to define a handler-specific contexts based on the incoming packet
	// and is run whenever a new packet arrives
	PacketContext func(context.Context, *packet.Packet) context.Context

	// UpdateContext is used to update a handler-specific context whenever the returned
	// Action from a handler is UPDATE
	UpdateContext func(context.Context, *Async) context.Context

	// OnClosed is a function run by the server whenever a connection is closed
	OnClosed func(*Async, error)

	// PreWrite is run by the server before a write happens
	PreWrite func()
}

// NewServer returns an uninitialized frisbee Server with the registered HandlerTable.
// The Start method must then be called to start the server and listen for connections.
func NewServer(addr string, handlerTable HandlerTable, opts ...Option) (*Server, error) {
	for i := uint16(0); i < RESERVED9; i++ {
		if _, ok := handlerTable[i]; ok {
			return nil, InvalidHandlerTable
		}
	}

	options := loadOptions(opts...)
	return &Server{
		addr:         addr,
		handlerTable: handlerTable,
		options:      options,
		shutdown:     atomic.NewBool(false),
		connections:  make(map[*Async]struct{}),
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
	go s.handleListener()

	return nil
}

func (s *Server) handleListener() {
	var newConn net.Conn
	var err error
	for {
		newConn, err = s.listener.Accept()
		if err != nil {
			if s.shutdown.Load() {
				s.wg.Done()
				return
			}
			s.Logger().Fatal().Err(err).Msg("error while accepting connection")
			s.wg.Done()
			return
		}
		s.wg.Add(1)
		go s.handleConn(newConn)
	}
}

func (s *Server) handlePacket(frisbeeConn *Async, connCtx context.Context) (err error) {
	var p *packet.Packet
	var outgoing *packet.Packet
	var action Action
	var handlerFunc Handler
	p, err = frisbeeConn.ReadPacket()
	if err != nil {
		return
	}
	if s.ConnContext != nil {
		connCtx = s.ConnContext(connCtx, frisbeeConn)
	}
	goto HANDLE
LOOP:
	p, err = frisbeeConn.ReadPacket()
	if err != nil {
		return
	}
HANDLE:
	handlerFunc = s.handlerTable[p.Metadata.Operation]
	if handlerFunc != nil {
		packetCtx := connCtx
		if s.PacketContext != nil {
			packetCtx = s.PacketContext(packetCtx, p)
		}
		outgoing, action = handlerFunc(packetCtx, p)
		if outgoing != nil && outgoing.Metadata.ContentLength == uint32(len(outgoing.Content.B)) {
			s.PreWrite()
			err = frisbeeConn.WritePacket(outgoing)
			if outgoing != p {
				packet.Put(outgoing)
			}
			packet.Put(p)
			if err != nil {
				return
			}
		} else {
			packet.Put(p)
		}
		switch action {
		case NONE:
		case UPDATE:
			if s.UpdateContext != nil {
				connCtx = s.UpdateContext(connCtx, frisbeeConn)
			}
		case CLOSE:
			return
		}
	} else {
		packet.Put(p)
	}
	goto LOOP
}

func (s *Server) handleConn(newConn net.Conn) {
	var err error
	switch v := newConn.(type) {
	case *net.TCPConn:
		err = v.SetKeepAlive(true)
		if err != nil {
			s.Logger().Error().Err(err).Msg("Error while setting TCP Keepalive")
			_ = v.Close()
			s.wg.Done()
			return
		}
		err = v.SetKeepAlivePeriod(s.options.KeepAlive)
		if err != nil {
			s.Logger().Error().Err(err).Msg("Error while setting TCP Keepalive Period")
			_ = v.Close()
			s.wg.Done()
			return
		}
	}

	frisbeeConn := NewAsync(newConn, s.Logger(), true)
	connCtx := s.BaseContext()

	s.connectionsMu.Lock()
	if s.shutdown.Load() {
		s.wg.Done()
		return
	}
	s.connections[frisbeeConn] = struct{}{}
	s.connectionsMu.Unlock()

	err = s.handlePacket(frisbeeConn, connCtx)
	_ = frisbeeConn.Close()
	s.OnClosed(frisbeeConn, err)
	s.connectionsMu.Lock()
	if !s.shutdown.Load() {
		delete(s.connections, frisbeeConn)
	}
	s.connectionsMu.Unlock()
	s.wg.Done()
}

// Logger returns the server's logger (useful for ServerRouter functions)
func (s *Server) Logger() *zerolog.Logger {
	return s.options.Logger
}

// Shutdown shuts down the frisbee server and kills all the goroutines and active connections
func (s *Server) Shutdown() error {
	s.shutdown.Store(true)
	s.connectionsMu.Lock()
	for c := range s.connections {
		_ = c.Close()
		delete(s.connections, c)
	}
	s.connectionsMu.Unlock()
	defer s.wg.Wait()
	return s.listener.Close()
}

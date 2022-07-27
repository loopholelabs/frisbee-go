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
	"net"
	"sync"
	"time"

	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
)

var (
	BaseContextNil = errors.New("BaseContext cannot be nil")
	OnClosedNil    = errors.New("OnClosed cannot be nil")
	PreWriteNil    = errors.New("PreWrite cannot be nil")
)

var (
	defaultBaseContext = context.Background

	defaultOnClosed = func(_ *Async, _ error) {}

	defaultPreWrite = func() {}
)

// Server accepts connections from frisbee Clients and can send and receive frisbee Packets
type Server struct {
	listener      net.Listener
	handlerTable  HandlerTable
	shutdown      *atomic.Bool
	options       *Options
	wg            sync.WaitGroup
	connections   map[*Async]struct{}
	connectionsMu sync.Mutex
	startedCh     chan struct{}

	// baseContext is used to define the base context for this Server and all incoming connections
	baseContext func() context.Context

	// onClosed is a function run by the server whenever a connection is closed
	onClosed func(*Async, error)

	// preWrite is run by the server before a write happens
	preWrite func()

	// ConnContext is used to define a connection-specific context based on the incoming connection
	// and is run whenever a new connection is opened
	ConnContext func(context.Context, *Async) context.Context

	// PacketContext is used to define a handler-specific contexts based on the incoming packet
	// and is run whenever a new packet arrives
	PacketContext func(context.Context, *packet.Packet) context.Context

	// UpdateContext is used to update a handler-specific context whenever the returned
	// Action from a handler is UPDATE
	UpdateContext func(context.Context, *Async) context.Context
}

// NewServer returns an uninitialized frisbee Server with the registered HandlerTable.
// The Start method must then be called to start the server and listen for connections.
func NewServer(handlerTable HandlerTable, opts ...Option) (*Server, error) {
	options := loadOptions(opts...)
	s := &Server{
		options:     options,
		shutdown:    atomic.NewBool(false),
		connections: make(map[*Async]struct{}),
		startedCh:   make(chan struct{}),
		baseContext: defaultBaseContext,
		onClosed:    defaultOnClosed,
		preWrite:    defaultPreWrite,
	}

	return s, s.SetHandlerTable(handlerTable)
}

// SetBaseContext sets the baseContext function for the server. If f is nil, it returns an error.
func (s *Server) SetBaseContext(f func() context.Context) error {
	if f == nil {
		return BaseContextNil
	}
	s.baseContext = f
	return nil
}

// SetOnClosed sets the onClosed function for the server. If f is nil, it returns an error.
func (s *Server) SetOnClosed(f func(*Async, error)) error {
	if f == nil {
		return OnClosedNil
	}
	s.onClosed = f
	return nil
}

// SetPreWrite sets the preWrite function for the server. If f is nil, it returns an error.
func (s *Server) SetPreWrite(f func()) error {
	if f == nil {
		return PreWriteNil
	}
	s.preWrite = f
	return nil
}

// SetHandlerTable sets the handler table for the server.
//
// This function should not be called once the server has started.
func (s *Server) SetHandlerTable(handlerTable HandlerTable) error {
	for i := uint16(0); i < RESERVED9; i++ {
		if _, ok := handlerTable[i]; ok {
			return InvalidHandlerTable
		}
	}

	if s.options.Heartbeat > time.Duration(0) {
		handlerTable[HEARTBEAT] = func(_ context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action) {
			outgoing = incoming
			return
		}
	}

	s.handlerTable = handlerTable
	return nil
}

// GetHandlerTable gets the handler table for the server.
//
// This function should not be called once the server has started.
func (s *Server) GetHandlerTable() HandlerTable {
	return s.handlerTable
}

// Start will start the frisbee server and its reactor goroutines
// to receive and handle incoming connections. If the baseContext, ConnContext,
// onClosed, OnShutdown, or preWrite functions have not been defined, it will
// use the default functions for these.
func (s *Server) Start(addr string) error {
	var err error
	if s.options.TLSConfig != nil {
		s.listener, err = tls.Listen("tcp", addr, s.options.TLSConfig)
	} else {
		s.listener, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return err
	}
	s.wg.Add(1)
	close(s.startedCh)
	return s.handleListener()
}

// started returns a channel that will be closed when the server has successfully started
//
// This is meant to only be used for testing purposes.
func (s *Server) started() <-chan struct{} {
	return s.startedCh
}

func (s *Server) handleListener() error {
	var backoff time.Duration
	for {
		newConn, err := s.listener.Accept()
		if err != nil {
			if s.shutdown.Load() {
				s.wg.Done()
				return nil
			}
			if ne, ok := err.(temporary); ok && ne.Temporary() {
				if backoff == 0 {
					backoff = minBackoff
				} else {
					backoff *= 2
				}
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				s.Logger().Warn().Err(err).Msgf("Temporary Accept Error, retrying in %s", backoff)
				time.Sleep(backoff)
				if s.shutdown.Load() {
					s.wg.Done()
					return nil
				}
				continue
			}
			s.wg.Done()
			return err
		}
		backoff = 0

		s.wg.Add(1)
		go func() {
			s.ServeConn(newConn)
			s.wg.Done()
		}()
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
		if outgoing != nil && outgoing.Metadata.ContentLength == uint32(len(*outgoing.Content)) {
			s.preWrite()
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

// ServeConn takes a TCP net.Conn and serves it using the Server
func (s *Server) ServeConn(newConn net.Conn) {
	var err error
	switch v := newConn.(type) {
	case *net.TCPConn:
		err = v.SetKeepAlive(true)
		if err != nil {
			s.Logger().Error().Err(err).Msg("Error while setting TCP Keepalive")
			_ = v.Close()
			return
		}
		err = v.SetKeepAlivePeriod(s.options.KeepAlive)
		if err != nil {
			s.Logger().Error().Err(err).Msg("Error while setting TCP Keepalive Period")
			_ = v.Close()
			return
		}
	}

	frisbeeConn := NewAsync(newConn, s.Logger())
	connCtx := s.baseContext()

	s.connectionsMu.Lock()
	if s.shutdown.Load() {
		return
	}
	s.connections[frisbeeConn] = struct{}{}
	s.connectionsMu.Unlock()

	err = s.handlePacket(frisbeeConn, connCtx)
	_ = frisbeeConn.Close()
	s.onClosed(frisbeeConn, err)
	s.connectionsMu.Lock()
	if !s.shutdown.Load() {
		delete(s.connections, frisbeeConn)
	}
	s.connectionsMu.Unlock()
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
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

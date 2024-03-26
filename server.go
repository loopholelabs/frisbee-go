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
	"errors"
	"net"
	"sync"
	"time"

	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
)

var (
	BaseContextNil   = errors.New("BaseContext cannot be nil")
	OnClosedNil      = errors.New("OnClosed cannot be nil")
	PreWriteNil      = errors.New("PreWrite cannot be nil")
	StreamHandlerNil = errors.New("StreamHandler cannot be nil")
	ListenerNil      = errors.New("Listener cannot be nil")
)

var (
	defaultBaseContext = context.Background

	defaultOnClosed = func(_ *Async, _ error) {}

	defaultPreWrite = func() {}

	defaultStreamHandler = func(stream *Stream) {
		_ = stream.Close()
	}
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
	concurrency   uint64
	limiter       chan struct{}

	// baseContext is used to define the base context for this Server and all incoming connections
	baseContext func() context.Context

	// onClosed is a function run by the server whenever a connection is closed
	onClosed func(*Async, error)

	// preWrite is run by the server before a write happens
	preWrite func()

	// streamHandler is used to handle incoming client-initiated streams on the server
	streamHandler func(*Stream)

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
		options:       options,
		shutdown:      atomic.NewBool(false),
		connections:   make(map[*Async]struct{}),
		startedCh:     make(chan struct{}),
		baseContext:   defaultBaseContext,
		onClosed:      defaultOnClosed,
		preWrite:      defaultPreWrite,
		streamHandler: defaultStreamHandler,
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

// SetStreamHandler sets the streamHandler function for the server. If f is nil, it returns an error.
func (s *Server) SetStreamHandler(f func(*Async, *Stream)) error {
	if f == nil {
		return StreamHandlerNil
	}
	s.streamHandler = func(stream *Stream) {
		f(stream.Conn(), stream)
	}
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

	s.handlerTable = handlerTable
	return nil
}

// GetHandlerTable gets the handler table for the server.
//
// This function should not be called once the server has started.
func (s *Server) GetHandlerTable() HandlerTable {
	return s.handlerTable
}

// SetConcurrency sets the maximum number of concurrent goroutines that will be created
// by the server to handle incoming packets.
//
// An important caveat of this is that handlers must always thread-safe if they share resources
// between connections. If the concurrency is set to a value != 1, then the handlers
// must also be thread-safe if they share resources per connection.
//
// This function should not be called once the server has started.
func (s *Server) SetConcurrency(concurrency uint64) {
	s.concurrency = concurrency
	if s.concurrency > 1 {
		s.limiter = make(chan struct{}, s.concurrency)
	}
}

// Start will start the frisbee server and its reactor goroutines
// to receive and handle incoming connections. If the baseContext, ConnContext,
// onClosed, OnShutdown, or preWrite functions have not been defined, it will
// use the default functions for these.
func (s *Server) Start(addr string) error {
	var listener net.Listener
	var err error
	if s.options.TLSConfig != nil {
		listener, err = tls.Listen("tcp", addr, s.options.TLSConfig)
	} else {
		listener, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return err
	}
	return s.StartWithListener(listener)
}

// StartWithListener will start the frisbee server and its reactor goroutines
// to receive and handle incoming connections with a given net.Listener. If the baseContext, ConnContext,
// onClosed, OnShutdown, or preWrite functions have not been defined, it will
// use the default functions for these.
func (s *Server) StartWithListener(listener net.Listener) error {
	if listener == nil {
		return ListenerNil
	}
	s.listener = listener
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

		s.ServeConn(newConn)
	}
}

func (s *Server) createHandler(conn *Async, closed *atomic.Bool, wg *sync.WaitGroup, ctx context.Context, cancel context.CancelFunc) func(*packet.Packet) {
	return func(p *packet.Packet) {
		handlerFunc := s.handlerTable[p.Metadata.Operation]
		if handlerFunc != nil {
			packetCtx := ctx
			if s.PacketContext != nil {
				packetCtx = s.PacketContext(packetCtx, p)
			}
			outgoing, action := handlerFunc(packetCtx, p)
			if outgoing != nil && outgoing.Metadata.ContentLength == uint32(len(*outgoing.Content)) {
				s.preWrite()
				err := conn.WritePacket(outgoing)
				if outgoing != p {
					packet.Put(outgoing)
				}
				packet.Put(p)
				if err != nil {
					_ = conn.Close()
					if closed.CompareAndSwap(false, true) {
						s.onClosed(conn, err)
					}
					cancel()
					wg.Done()
					return
				}
			} else {
				packet.Put(p)
			}
			switch action {
			case NONE:
			case CLOSE:
				_ = conn.Close()
				if closed.CompareAndSwap(false, true) {
					s.onClosed(conn, nil)
				}
				cancel()
			}
		} else {
			packet.Put(p)
		}
		wg.Done()
	}
}

func (s *Server) handleSinglePacket(frisbeeConn *Async, connCtx context.Context) {
	var p *packet.Packet
	var outgoing *packet.Packet
	var action Action
	var handlerFunc Handler
	var err error
	p, err = frisbeeConn.ReadPacket()
	if err != nil {
		_ = frisbeeConn.Close()
		s.onClosed(frisbeeConn, err)
		return
	}
	for {
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
					_ = frisbeeConn.Close()
					s.onClosed(frisbeeConn, err)
					return
				}
			} else {
				packet.Put(p)
			}
			switch action {
			case NONE:
			case CLOSE:
				_ = frisbeeConn.Close()
				s.onClosed(frisbeeConn, nil)
				return
			}
		} else {
			packet.Put(p)
		}
		p, err = frisbeeConn.ReadPacket()
		if err != nil {
			_ = frisbeeConn.Close()
			s.onClosed(frisbeeConn, err)
			return
		}
	}
}

func (s *Server) handleUnlimitedPacket(frisbeeConn *Async, connCtx context.Context) {
	p, err := frisbeeConn.ReadPacket()
	if err != nil {
		_ = frisbeeConn.Close()
		s.onClosed(frisbeeConn, err)
		return
	}
	wg := new(sync.WaitGroup)
	closed := atomic.NewBool(false)
	connCtx, cancel := context.WithCancel(connCtx)
	handle := s.createHandler(frisbeeConn, closed, wg, connCtx, cancel)
	for {
		wg.Add(1)
		go handle(p)
		p, err = frisbeeConn.ReadPacket()
		if err != nil {
			_ = frisbeeConn.Close()
			if closed.CompareAndSwap(false, true) {
				s.onClosed(frisbeeConn, err)
			}
			cancel()
			wg.Wait()
			return
		}
	}
}

func (s *Server) handleLimitedPacket(frisbeeConn *Async, connCtx context.Context) {
	p, err := frisbeeConn.ReadPacket()
	if err != nil {
		_ = frisbeeConn.Close()
		s.onClosed(frisbeeConn, err)
		return
	}
	wg := new(sync.WaitGroup)
	closed := atomic.NewBool(false)
	connCtx, cancel := context.WithCancel(connCtx)
	handler := s.createHandler(frisbeeConn, closed, wg, connCtx, cancel)
	handle := func(p *packet.Packet) {
		handler(p)
		<-s.limiter
	}
	for {
		select {
		case s.limiter <- struct{}{}:
			wg.Add(1)
			go handle(p)
			p, err = frisbeeConn.ReadPacket()
			if err != nil {
				_ = frisbeeConn.Close()
				if closed.CompareAndSwap(false, true) {
					s.onClosed(frisbeeConn, err)
				}
				cancel()
				wg.Wait()
				return
			}
		case <-connCtx.Done():
			_ = frisbeeConn.Close()
			if closed.CompareAndSwap(false, true) {
				s.onClosed(frisbeeConn, err)
			}
			wg.Wait()
			return
		}
	}
}

// ServeConn takes a net.Conn and starts a goroutine to handle it using the Server.
func (s *Server) ServeConn(conn net.Conn) {
	s.wg.Add(1)
	go s.serveConn(conn)
}

// serveConn takes a net.Conn and serves it using the Server
// and assumes that the server's wait group has been incremented by 1.
func (s *Server) serveConn(newConn net.Conn) {
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

	frisbeeConn := NewAsync(newConn, s.Logger(), s.streamHandler)
	connCtx := s.baseContext()
	s.connectionsMu.Lock()
	if s.shutdown.Load() {
		s.wg.Done()
		return
	}
	s.connections[frisbeeConn] = struct{}{}
	s.connectionsMu.Unlock()
	if s.ConnContext != nil {
		connCtx = s.ConnContext(connCtx, frisbeeConn)
	}
	if s.concurrency == 0 {
		s.handleUnlimitedPacket(frisbeeConn, connCtx)
	} else if s.concurrency == 1 {
		s.handleSinglePacket(frisbeeConn, connCtx)
	} else {
		s.handleLimitedPacket(frisbeeConn, connCtx)
	}
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
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

package handler

import (
	"context"
	"encoding/binary"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/codec"
	"github.com/loophole-labs/frisbee/internal/conn"
	"github.com/loophole-labs/frisbee/internal/log"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/panjf2000/ants/v2"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	"github.com/rs/zerolog"
	"time"
)

type Handler struct {
	*gnet.EventServer
	addr               string
	multicore          bool
	async              bool
	workerPool         *goroutine.Pool
	codec              *codec.ICodec
	router             frisbee.ServerRouter
	logger             log.Logger
	started            chan struct{}
	UserOnInitComplete func() frisbee.Action
	UserOnOpened       func(c frisbee.Conn) frisbee.Action
	UserOnClosed       func(c frisbee.Conn, err error) frisbee.Action
	UserOnShutdown     func()
	UserPreWrite       func()
	UserTick           func() (time.Duration, frisbee.Action)
}

func (handler *Handler) OnInitComplete(_ gnet.Server) gnet.Action {
	handler.started <- struct{}{}
	return gnet.Action(handler.UserOnInitComplete())
}

func (handler *Handler) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	action = gnet.Action(handler.UserOnOpened(frisbee.Conn{Conn: conn.Convert(c)}))
	return
}

func (handler *Handler) OnClosed(c gnet.Conn, err error) gnet.Action {
	return gnet.Action(handler.UserOnClosed(frisbee.Conn{Conn: conn.Convert(c)}, err))
}

func (handler *Handler) PreWrite() {
	handler.UserPreWrite()
}

func (handler *Handler) Tick() (delay time.Duration, action gnet.Action) {
	delay, frisbeeAction := handler.UserTick()
	action = gnet.Action(frisbeeAction)
	return
}

func (handler *Handler) OnShutdown(_ gnet.Server) {
	handler.UserOnShutdown()
}

func (handler *Handler) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	id := binary.BigEndian.Uint32(frame)
	packet := handler.codec.Packets[id]
	handler.logger.Debugf("Received message %d", id)
	handlerFunc := handler.router[packet.Message.Operation]
	if handlerFunc != nil {
		message, output, frisbeeAction := handlerFunc(frisbee.Conn{Conn: conn.Convert(c)}, frisbee.Message(*packet.Message), packet.Content)
		action = gnet.Action(frisbeeAction)
		if message != nil && message.ContentLength == uint32(len(output)) {
			encodedMessage, err := protocol.EncodeV0(message.Id, message.Operation, message.Routing, message.ContentLength)
			if err != nil {
				return
			}
			if message.ContentLength > 0 {
				if handler.async {
					_ = handler.workerPool.Submit(func() {
						_ = c.AsyncWrite(append(encodedMessage[:], output...))
					})
					return nil, action
				}
				return append(encodedMessage[:], output...), action
			}
			return encodedMessage[:], action
		}
		return
	}
	return

}

func (handler *Handler) Stop() error {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	return gnet.Stop(ctx, handler.addr)
}

func StartHandler(started chan struct{}, addr string, multicore bool, async bool, loops int, keepAlive time.Duration, logger *zerolog.Logger, router frisbee.ServerRouter, UserOnInitComplete func() frisbee.Action, UserOnOpened func(c frisbee.Conn) frisbee.Action, UserOnClosed func(c frisbee.Conn, err error) frisbee.Action, UserOnShutdown func(), UserPreWrite func(), UserTick func() (time.Duration, frisbee.Action)) *Handler {
	icCodec := &codec.ICodec{
		Packets: make(map[uint32]*codec.Packet),
	}

	options := ants.Options{ExpiryDuration: time.Second * 10, Nonblocking: false}
	defaultAntsPool, _ := ants.NewPool(1<<18, ants.WithOptions(options))

	if UserOnClosed == nil {
		UserOnClosed = func(_ frisbee.Conn, _ error) frisbee.Action {
			return frisbee.None
		}
	}

	if UserOnOpened == nil {
		UserOnOpened = func(_ frisbee.Conn) frisbee.Action {
			return frisbee.None
		}
	}

	if UserOnInitComplete == nil {
		UserOnInitComplete = func() frisbee.Action {
			return frisbee.None
		}
	}

	if UserOnShutdown == nil {
		UserOnShutdown = func() {}
	}

	if UserPreWrite == nil {
		UserPreWrite = func() {}
	}

	if UserTick == nil {
		UserTick = func() (delay time.Duration, action frisbee.Action) {
			return
		}
	}

	handler := &Handler{
		addr:               addr,
		multicore:          multicore,
		async:              async,
		codec:              icCodec,
		workerPool:         defaultAntsPool,
		router:             router,
		logger:             log.Convert(logger),
		started:            started,
		UserOnInitComplete: UserOnInitComplete,
		UserOnOpened:       UserOnOpened,
		UserOnShutdown:     UserOnShutdown,
		UserOnClosed:       UserOnClosed,
		UserPreWrite:       UserPreWrite,
		UserTick:           UserTick,
	}
	go func() {
		err := gnet.Serve(
			handler,
			addr,
			gnet.WithMulticore(multicore),
			gnet.WithNumEventLoop(loops),
			gnet.WithTCPKeepAlive(keepAlive),
			gnet.WithLogger(log.Convert(logger)),
			gnet.WithCodec(icCodec),
			gnet.WithLockOSThread(true),
			gnet.WithReusePort(true))
		if err != nil {
			panic(err)
		}
	}()

	return handler
}

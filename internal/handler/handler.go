package handler

import (
	"context"
	"encoding/binary"
	"github.com/loophole-labs/frisbee"
	"github.com/loophole-labs/frisbee/internal/codec"
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
	addr       string
	multicore  bool
	async      bool
	workerPool *goroutine.Pool
	codec      *codec.ICodec
	router     frisbee.Router
	logger     log.Logger
	started    chan struct{}
}

func (handler *Handler) OnInitComplete(_ gnet.Server) (action gnet.Action) {
	handler.started <- struct{}{}
	handler.logger.Infof("Startup Complete")
	return 0
}

func (handler *Handler) OnOpened(_ gnet.Conn) (out []byte, action gnet.Action) {
	handler.logger.Infof("Client Connected")
	return nil, 0
}

func (handler *Handler) OnShutdown(_ gnet.Server) {
}

func (handler *Handler) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	id := binary.BigEndian.Uint32(frame)
	packet := handler.codec.Packets[id]
	handlerFunc := handler.router[packet.Message.Operation]
	if handlerFunc != nil {
		message, output, frisbeeAction := handlerFunc(frisbee.Message(*packet.Message), packet.Content)

		action = gnet.Action(frisbeeAction)
		if message != nil && message.ContentLength == uint32(len(output)) {
			encodedMessage, err := protocol.EncodeV0(message.Id, message.Operation, message.Routing, message.ContentLength)
			if err != nil {
				return
			}
			if message.ContentLength > 0 {
				if handler.async {
					if handler.workerPool.Free() > 0 {
						_ = handler.workerPool.Submit(func() {
							_ = c.AsyncWrite(append(encodedMessage[:], output...))
						})
					} else {
						go func() {
							_ = handler.workerPool.Submit(func() {
								_ = c.AsyncWrite(append(encodedMessage[:], output...))
							})
						}()
					}
					return nil, action
				}
				return append(encodedMessage[:], output...), action
			}
			return encodedMessage[:], action
		}
		return
	}
	out = nil
	return

}

func (handler *Handler) Stop() (err error) {
	err = gnet.Stop(context.Background(), handler.addr)
	return
}

func StartHandler(started chan struct{}, addr string, multicore bool, async bool, loops int, keepAlive time.Duration, logger *zerolog.Logger, router frisbee.Router) *Handler {
	icCodec := &codec.ICodec{
		Packets: make(map[uint32]*codec.Packet),
	}

	options := ants.Options{ExpiryDuration: time.Second * 10, Nonblocking: false}
	defaultAntsPool, _ := ants.NewPool(1<<18, ants.WithOptions(options))

	handler := &Handler{
		addr:       addr,
		multicore:  multicore,
		async:      async,
		codec:      icCodec,
		workerPool: defaultAntsPool,
		router:     router,
		logger:     log.Convert(logger),
		started:    started,
	}
	go func() {
		gnetError := gnet.Serve(
			handler,
			addr,
			gnet.WithMulticore(multicore),
			gnet.WithNumEventLoop(loops),
			gnet.WithTCPKeepAlive(keepAlive),
			gnet.WithLogger(log.Convert(logger)),
			gnet.WithCodec(icCodec))
		if gnetError != nil {
			panic(gnetError)
		}
	}()

	return handler
}

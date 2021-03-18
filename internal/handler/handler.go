package handler

import (
	"encoding/binary"
	"github.com/loophole-labs/frisbee/internal/codec"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/panjf2000/ants/v2"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	"github.com/sirupsen/logrus"
	"time"
)

type HandleFunc func(message protocol.MessageV0, content []byte) ([]byte, int)
type MessageMap map[uint16]HandleFunc

type Handler struct {
	*gnet.EventServer
	addr       string
	multicore  bool
	async      bool
	workerPool *goroutine.Pool
	codec      *codec.ICodec
	router     MessageMap
	started    chan struct{}
}

func (handler *Handler) OnInitComplete(_ gnet.Server) (action gnet.Action) {
	handler.started <- struct{}{}
	return 0
}

func (handler *Handler) OnOpened(_ gnet.Conn) (out []byte, action gnet.Action) {
	return nil, 0
}

func (handler *Handler) OnShutdown(_ gnet.Server) {
}

func (handler *Handler) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	id := binary.BigEndian.Uint32(frame)
	packet := handler.codec.Packets[id]
	handlerFunc := handler.router[packet.Message.Operation]
	if handlerFunc != nil {
		out, actionInt := handlerFunc(*packet.Message, packet.Content)
		action = gnet.Action(actionInt)
		if handler.async && out != nil {
			if handler.workerPool.Free() > 0 {
				_ = handler.workerPool.Submit(func() {
					_ = c.AsyncWrite(out)
				})
			} else {
				go func() {
					_ = handler.workerPool.Submit(func() {
						_ = c.AsyncWrite(out)
					})
				}()
			}
			return nil, action
		}
		return out, action
	}
	out = nil
	return

}

func StartHandler(started chan struct{}, addr string, multicore bool, async bool, loops int, keepAlive time.Duration, log *logrus.Logger, router MessageMap) {
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
		started:    started,
	}
	go func() {
		err := gnet.Serve(
			handler,
			addr,
			gnet.WithMulticore(multicore),
			gnet.WithNumEventLoop(loops),
			gnet.WithTCPKeepAlive(keepAlive),
			gnet.WithLogger(log),
			gnet.WithCodec(icCodec))

		if err != nil {
			panic(err)
		}
	}()
}

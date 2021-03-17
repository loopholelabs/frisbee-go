package router

import (
	"encoding/binary"
	"github.com/loophole-labs/frisbee/internal/codec"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"time"
)

type RouteFunc func(message protocol.MessageV0, content []byte) ([]byte, int)
type MessageMap map[uint16]RouteFunc

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

func (handler *Handler) OnInitComplete(server gnet.Server) (action gnet.Action) {
	handler.started <- struct{}{}
	return 0
}

func (handler *Handler) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	return nil, 0
}

func (handler *Handler) OnShutdown(svr gnet.Server) {
}

func (handler *Handler) React(frame []byte, _ gnet.Conn) (out []byte, action gnet.Action) {
	id := binary.BigEndian.Uint32(frame)
	packet := handler.codec.Packets[id]
	handlerFunc := handler.router[packet.Message.Operation]
	if handlerFunc != nil {
		out, actionInt := handlerFunc(*packet.Message, packet.Content)
		action = gnet.Action(actionInt)
		return out, action
	}
	out = nil
	return

}

func StartServer(started chan struct{}, addr string, multicore bool, async bool, router MessageMap) {
	icCodec := &codec.ICodec{
		Packets: make(map[uint32]*codec.Packet),
	}
	server := &Handler{
		addr:       addr,
		multicore:  multicore,
		async:      async,
		codec:      icCodec,
		workerPool: goroutine.Default(),
		router:     router,
		started:    started,
	}
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)
	go func() {
		err := gnet.Serve(server, addr, gnet.WithMulticore(multicore), gnet.WithTCPKeepAlive(time.Minute*5), gnet.WithCodec(icCodec), gnet.WithLogger(logger), gnet.WithNumEventLoop(16))
		if err != nil {
			panic(err)
		}
	}()
}

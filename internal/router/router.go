package router

import (
	"encoding/binary"
	"github.com/loophole-labs/frisbee/internal/codec"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	"log"
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
}

func (handler *Handler) React(frame []byte, _ gnet.Conn) (out []byte, action gnet.Action) {
	id := binary.BigEndian.Uint16(frame)
	packet := handler.codec.Packets[id]
	handlerFunc := handler.router[packet.Message.Operation]
	if handlerFunc != nil {
		out, actionInt := handlerFunc(*packet.Message, *packet.Content)
		action = gnet.Action(actionInt)
		return out, action
	} else {
		log.Printf("Operation 0x%x was invalid", packet.Message.Operation)
	}
	out = nil
	return

}

func StartServer(addr string, multicore bool, async bool, router MessageMap) {
	icCodec := &codec.ICodec{
		Packets: make(map[uint16]*codec.Packet),
	}
	server := &Handler{
		addr:       addr,
		multicore:  multicore,
		async:      async,
		codec:      icCodec,
		workerPool: goroutine.Default(),
		router:     router,
	}
	go gnet.Serve(server, addr, gnet.WithMulticore(multicore), gnet.WithTCPKeepAlive(time.Minute*5), gnet.WithCodec(icCodec), gnet.WithNumEventLoop(16), gnet.WithLoadBalancing(gnet.RoundRobin))
}

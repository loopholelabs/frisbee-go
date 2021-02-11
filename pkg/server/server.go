package server

import (
	"time"

	"github.com/loophole-labs/frisbee/pkg/codec"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
)

type gnetServer struct {
	*gnet.EventServer
	addr       string
	multicore  bool
	async      bool
	codec      gnet.ICodec
	workerPool *goroutine.Pool
}

func (server *gnetServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {

	//message, _ := protocol.DecodeV0(encodedData, true, false)

	//log.Printf("Frame: %s", string(frame))

	//response := codec.MessageV0{
	//	Operation: protocol.MessagePong,
	//	Routing:   0,
	//}
	//
	//c.SetContext(response)
	//
	//if server.async {
	//	data := append([]byte{}, frame...)
	//	_ = server.workerPool.Submit(func() {
	//		c.AsyncWrite(data)
	//	})
	//	return
	//}
	out = nil
	return
}

func StartServer(addr string, multicore, async bool, icCodec gnet.ICodec) {
	icCodec = &codec.MessageV0{}
	server := &gnetServer{addr: addr, multicore: multicore, async: async, codec: icCodec, workerPool: goroutine.Default()}
	go gnet.Serve(server, addr, gnet.WithMulticore(multicore), gnet.WithTCPKeepAlive(time.Minute*5), gnet.WithCodec(icCodec), gnet.WithNumEventLoop(16), gnet.WithLoadBalancing(gnet.RoundRobin))

}

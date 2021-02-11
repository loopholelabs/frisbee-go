package main

import (
	"fmt"
	"log"
	"time"

	"github.com/loophole-labs/frisbee/internal/protocol"
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

func (server *gnetServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("Test codec server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
	return
}

func (server *gnetServer) React(encodedData []byte, c gnet.Conn) (out []byte, action gnet.Action) {

	message, _ := protocol.DecodeV0(encodedData, false, false)

	log.Printf("Operation: %x, Routing %x, Content: %s", message.Operation, message.Routing, string(message.Content))

	response := codec.MessageV0{
		Operation: protocol.MessagePong,
		Routing:   0,
	}

	c.SetContext(response)

	if server.async {
		data := append([]byte{}, message.Content...)
		_ = server.workerPool.Submit(func() {
			c.AsyncWrite(data)
		})
		return
	}
	out = message.Content
	return
}

func startServer(addr string, multicore, async bool, icCodec gnet.ICodec) {
	var err error
	icCodec = &codec.MessageV0{}
	server := &gnetServer{addr: addr, multicore: multicore, async: async, codec: icCodec, workerPool: goroutine.Default()}
	err = gnet.Serve(server, addr, gnet.WithMulticore(multicore), gnet.WithTCPKeepAlive(time.Minute*5), gnet.WithCodec(icCodec))
	if err != nil {
		panic(err)
	}
}

func main() {
	addr := fmt.Sprintf("tcp://:8192")
	startServer(addr, true, false, nil)
}

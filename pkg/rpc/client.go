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

package rpc

import (
	"github.com/loopholelabs/frisbee/internal/utils"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	clientStruct  = "Client"
	frisbeeClient = "*frisbee.Client"
)

func writeClientHandlers(f File, services protoreflect.ServiceDescriptors) {
	operationIndex := 0
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			method := service.Methods().Get(j)
			f.P(tab, "table[", operationIndex+operationOffset, "] = ", handlerSignature, " {")
			f.P(tab, tab, tab, "c.inflight", utils.CamelCase(string(method.Name())), "Mu.RLock()")
			f.P(tab, tab, tab, "if ch, ok := c.inflight", utils.CamelCase(string(method.Name())), "[incoming.Metadata.Id]; ok {")
			f.P(tab, tab, tab, tab, "c.inflight", utils.CamelCase(string(method.Name())), "Mu.RUnlock()")
			f.P(tab, tab, tab, tab, "res := New", utils.CamelCase(string(method.Output().FullName())), "()")
			f.P(tab, tab, tab, tab, "res.Decode(incoming)")
			f.P(tab, tab, tab, tab, "ch <- res")
			f.P(tab, tab, tab, "} else {")
			f.P(tab, tab, tab, tab, "c.inflight", utils.CamelCase(string(method.Name())), "Mu.RUnlock()")
			f.P(tab, tab, tab, "}")
			f.P(tab, tab, tab, "return")
			f.P(tab, tab, "}")
			operationIndex++
		}
	}
}

func writeClientMethod(f File, method protoreflect.MethodDescriptor, operation int) {
	f.P("func (c *Client) ", utils.CamelCase(string(method.Name())), "(ctx context.Context, req *", utils.CamelCase(string(method.Input().FullName())), ") (res *", utils.CamelCase(string(method.Output().FullName())), ", err error) {")
	f.P(tab, "ch := make(chan *", utils.CamelCase(string(method.Output().FullName())), ", 1)")
	f.P(tab, "p := packet.Get()")
	f.P(tab, "p.Metadata.Operation = ", operation)
	f.P("LOOP:")
	f.P(tab, "p.Metadata.Id = c.next", utils.CamelCase(string(method.Name())), ".Load().(uint16)")
	f.P(tab, "if !c.next", utils.CamelCase(string(method.Name())), ".CompareAndSwap(p.Metadata.Id, p.Metadata.Id+1) {")
	f.P(tab, tab, "goto LOOP")
	f.P(tab, "}")
	f.P(tab, "req.Encode(p)")
	f.P(tab, "p.Metadata.ContentLength = uint32(len(p.Content.B))")
	f.P(tab, "c.inflight", utils.CamelCase(string(method.Name())), "Mu.Lock()")
	f.P(tab, "c.inflight", utils.CamelCase(string(method.Name())), "[p.Metadata.Id] = ch")
	f.P(tab, "c.inflight", utils.CamelCase(string(method.Name())), "Mu.Unlock()")
	f.P(tab, "err = c.Client.WritePacket(p)")
	f.P(tab, "if err != nil {")
	f.P(tab, tab, "packet.Put(p)")
	f.P(tab, tab, "return")
	f.P(tab, "}")
	f.P(tab, "select {")
	f.P(tab, "case res = <- ch:")
	f.P(tab, tab, "err = res.error")
	f.P(tab, "case <- ctx.Done():")
	f.P(tab, tab, "err = ctx.Err()")
	f.P(tab, "}")
	f.P(tab, "c.inflight", utils.CamelCase(string(method.Name())), "Mu.Lock()")
	f.P(tab, "delete(c.inflight", utils.CamelCase(string(method.Name())), ", p.Metadata.Id)")
	f.P(tab, "c.inflight", utils.CamelCase(string(method.Name())), "Mu.Unlock()")
	f.P(tab, "packet.Put(p)")
	f.P(tab, "return")
	f.P("}")
	f.P()
}

func writeClientIgnoreMethod(f File, method protoreflect.MethodDescriptor, operation int) {
	f.P("func (c *Client) ", utils.CamelCase(string(method.Name())), "Ignore(ctx context.Context, req *", utils.CamelCase(string(method.Input().FullName())), ") (err error) {")
	f.P(tab, "p := packet.Get()")
	f.P(tab, "p.Metadata.Operation = ", operation)
	f.P("LOOP:")
	f.P(tab, "p.Metadata.Id = c.next", utils.CamelCase(string(method.Name())), ".Load().(uint16)")
	f.P(tab, "if !c.next", utils.CamelCase(string(method.Name())), ".CompareAndSwap(p.Metadata.Id, p.Metadata.Id+1) {")
	f.P(tab, tab, "goto LOOP")
	f.P(tab, "}")
	f.P(tab, "req.ignore = true")
	f.P(tab, "req.Encode(p)")
	f.P(tab, "p.Metadata.ContentLength = uint32(len(p.Content.B))")
	f.P(tab, "err = c.Client.WritePacket(p)")
	f.P(tab, "packet.Put(p)")
	f.P(tab, "return")
	f.P("}")
	f.P()
}

func writeClientMethods(f File, services protoreflect.ServiceDescriptors) {
	operationIndex := 0
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			method := service.Methods().Get(j)
			writeClientMethod(f, method, operationIndex)
			writeClientIgnoreMethod(f, method, operationIndex)
			operationIndex++
		}
	}
}

func writeClient(f File, services protoreflect.ServiceDescriptors) {
	f.P(typeOpen, clientStruct, structOpen)
	f.P(tab, frisbeeClient)
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			method := service.Methods().Get(j)
			f.P(tab, "next", utils.CamelCase(string(method.Name())), " atomic.Value")
			f.P(tab, "inflight", utils.CamelCase(string(method.Name())), "Mu sync.RWMutex")
			f.P(tab, "inflight", utils.CamelCase(string(method.Name())), space, "map[uint16]chan *", utils.CamelCase(string(method.Output().FullName())))
		}
	}
	f.P(typeClose)
	f.P()

	f.P("func NewClient(addr string, tlsConfig *tls.Config, logger *zerolog.Logger) (*Client, error) {")
	f.P(tab, "c := new(Client)")
	f.P(tab, "table := make(frisbee.HandlerTable)")
	writeClientHandlers(f, services)

	f.P(tab, "var err error")
	f.P(tab, "if tlsConfig != nil {")
	f.P(tab, tab, "c.Client, err = frisbee.NewClient(addr, table, context.Background(), frisbee.WithTLS(tlsConfig), frisbee.WithLogger(logger))")
	f.P(tab, tab, "if err != nil {")
	f.P(tab, tab, tab, "return nil, err")
	f.P(tab, tab, "}")
	f.P(tab, "} else {")
	f.P(tab, tab, "c.Client, err = frisbee.NewClient(addr, table, context.Background(), frisbee.WithLogger(logger))")
	f.P(tab, tab, "if err != nil {")
	f.P(tab, tab, tab, "return nil, err")
	f.P(tab, tab, "}")
	f.P(tab, "}")
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			method := service.Methods().Get(j)
			f.P(tab, "c.next", utils.CamelCase(string(method.Name())), ".Store(uint16(0))")
			f.P(tab, "c.inflight", utils.CamelCase(string(method.Name())), " = make(map[uint16]chan *", utils.CamelCase(string(method.Output().FullName())), ")")
		}
	}
	f.P(tab, "return c, nil")
	f.P(typeClose)
	f.P()
	writeClientMethods(f, services)
}

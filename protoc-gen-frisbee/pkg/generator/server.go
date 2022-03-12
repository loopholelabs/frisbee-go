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

package generator

import (
	"github.com/loopholelabs/frisbee/protoc-gen-frisbee/internal/utils"
	"google.golang.org/protobuf/reflect/protoreflect"
	"strings"
)

const (
	serverStruct  = "Server"
	frisbeeServer = "*frisbee.Server"
)

func writeServerHandlers(f File, services protoreflect.ServiceDescriptors) {
	operationIndex := 0
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			method := service.Methods().Get(j)
			f.P(tab, "table[", operationIndex+operationOffset, "] = ", handlerSignature, " {")
			f.P(tab, tab, tab, "req := New", utils.CamelCase(string(method.Input().FullName())), "()")
			f.P(tab, tab, tab, "err := req.Decode(incoming)")
			f.P(tab, tab, tab, "if err == nil {")
			f.P(tab, tab, tab, tab, "if req.ignore {")
			f.P(tab, tab, tab, tab, tab, utils.FirstLowerCase(utils.CamelCase(string(service.Name()))), period,
				utils.CamelCase(string(method.Name())), "(ctx, req)")
			f.P(tab, tab, tab, tab, "} else {")
			f.P(tab, tab, tab, tab, tab, "var res *", utils.CamelCase(string(method.Output().FullName())))
			f.P(tab, tab, tab, tab, tab, "outgoing = incoming")
			f.P(tab, tab, tab, tab, tab, "outgoing.Content.Reset()")
			f.P(tab, tab, tab, tab, tab, "res, err = ", utils.FirstLowerCase(utils.CamelCase(string(service.Name()))), period,
				utils.CamelCase(string(method.Name())), "(ctx, req)")
			f.P(tab, tab, tab, tab, tab, "if err != nil {")
			f.P(tab, tab, tab, tab, tab, tab, "res.Error(outgoing, err)")
			f.P(tab, tab, tab, tab, tab, "} else {")
			f.P(tab, tab, tab, tab, tab, tab, "res.Encode(outgoing)")
			f.P(tab, tab, tab, tab, tab, "}")
			f.P(tab, tab, tab, tab, tab, "outgoing.Metadata.ContentLength = uint32(len(outgoing.Content.B))")
			f.P(tab, tab, tab, tab, "}")
			f.P(tab, tab, tab, "}")
			f.P(tab, tab, tab, "return")
			f.P(tab, tab, "}")
			operationIndex++
		}
	}
}

func writeServer(f File, services protoreflect.ServiceDescriptors) {
	f.P(typeOpen, serverStruct, structOpen)
	f.P(tab, frisbeeServer)
	f.P(typeClose)
	f.P()
	builder := new(strings.Builder)
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		serviceName := utils.CamelCase(string(service.Name()))
		builder.WriteString(utils.FirstLowerCase(serviceName))
		builder.WriteString(space)
		builder.WriteString(serviceName)
		builder.WriteString(comma)
		builder.WriteString(space)
	}
	serverFields := builder.String()
	serverFields = serverFields[:len(serverFields)-2]
	f.P("func NewServer(", serverFields, ", listenAddr string, tlsConfig *tls.Config, logger *zerolog.Logger) (*Server, error) {")
	f.P(tab, "table := make(frisbee.HandlerTable)")
	writeServerHandlers(f, services)

	f.P(tab, "var s *frisbee.Server")
	f.P(tab, "var err error")
	f.P(tab, "if tlsConfig != nil {")
	f.P(tab, tab, "s, err = frisbee.NewServer(listenAddr, table, frisbee.WithTLS(tlsConfig), frisbee.WithLogger(logger))")
	f.P(tab, tab, "if err != nil {")
	f.P(tab, tab, tab, "return nil, err")
	f.P(tab, tab, "}")
	f.P(tab, "} else {")
	f.P(tab, tab, "s, err = frisbee.NewServer(listenAddr, table, frisbee.WithLogger(logger))")
	f.P(tab, tab, "if err != nil {")
	f.P(tab, tab, tab, "return nil, err")
	f.P(tab, tab, "}")
	f.P("}")
	f.P(tab, "return &Server{")
	f.P(tab, tab, "Server: s,")
	f.P(tab, "}, nil")
	f.P(typeClose)
}

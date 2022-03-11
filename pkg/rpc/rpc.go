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
	"github.com/loopholelabs/frisbee/pkg/rpc/version"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

type Generator struct {
	options *protogen.Options
}

func New() *Generator {
	return &Generator{
		options: &protogen.Options{
			ParamFunc:         func(name string, value string) error { return nil },
			ImportRewriteFunc: func(path protogen.GoImportPath) protogen.GoImportPath { return path },
		},
	}
}

func (g *Generator) UnmarshalRequest(buf []byte) (*pluginpb.CodeGeneratorRequest, error) {
	req := new(pluginpb.CodeGeneratorRequest)
	return req, proto.Unmarshal(buf, req)
}

func (g *Generator) MarshalResponse(res *pluginpb.CodeGeneratorResponse) ([]byte, error) {
	return proto.Marshal(res)
}

func (g *Generator) Generate(req *pluginpb.CodeGeneratorRequest) (res *pluginpb.CodeGeneratorResponse, err error) {
	plugin, err := g.options.New(req)
	if err != nil {
		return nil, err
	}

	for _, f := range plugin.Files {
		if !f.Generate {
			continue
		}
		genFile := plugin.NewGeneratedFile(fileName(f.GeneratedFilenamePrefix), f.GoImportPath)
		writeComment(genFile, version.Version, f.Desc.Path())
		if f.Desc.Package().Name() == "" {
			writePackage(genFile, f.GoPackageName)
		} else {
			writePackage(genFile, protogen.GoPackageName(f.Desc.Package().Name()))
		}
		writeImports(genFile, requiredImports)
		writeErrors(genFile)

		for i := 0; i < f.Desc.Enums().Len(); i++ {
			enum := f.Desc.Enums().Get(i)
			writeEnums(genFile, string(enum.FullName()), enum.Values())
		}

		for i := 0; i < f.Desc.Messages().Len(); i++ {
			message := f.Desc.Messages().Get(i)
			message.Options().ProtoReflect()
			for i := 0; i < message.Enums().Len(); i++ {
				enum := message.Enums().Get(i)
				writeEnums(genFile, string(enum.FullName()), enum.Values())
			}
			writeStructs(genFile, string(message.FullName()), message.Fields(), message.Messages())
		}

		for i := 0; i < f.Desc.Services().Len(); i++ {
			service := f.Desc.Services().Get(i)
			writeInterface(genFile, string(service.Name()), service.Methods())
		}

		writeServer(genFile, f.Desc.Services())
		writeClient(genFile, f.Desc.Services())
	}

	return plugin.Response(), nil
}

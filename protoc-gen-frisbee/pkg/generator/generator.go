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
	"github.com/loopholelabs/frisbee/protoc-gen-frisbee/internal/version"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
	"text/template"
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

	temp := template.Must(template.New("main").Funcs(template.FuncMap{
		"CamelCase":         utils.CamelCaseFN,
		"CamelCaseN":        utils.CamelCaseN,
		"MakeIterable":      utils.MakeIterable,
		"Counter":           utils.Counter,
		"FirstLowerCase":    utils.FirstLowerCase,
		"FirstLowerCaseN":   utils.FirstLowerCaseN,
		"FindValue":         findValue,
		"GetKind":           getKind,
		"GetLUTEncoder":     getLUTEncoder,
		"GetLUTDecoder":     getLUTDecoder,
		"GetEncodingFields": getEncodingFields,
		"GetDecodingFields": getDecodingFields,
		"GetKindLUT":        getKindLUT,
		"GetServerFields":   getServerFields,
	}).ParseGlob("protoc-gen-frisbee/templates/*"))

	for _, f := range plugin.Files {
		if !f.Generate {
			continue
		}
		genFile := plugin.NewGeneratedFile(fileName(f.GeneratedFilenamePrefix), f.GoImportPath)

		packageName := string(f.Desc.Package().Name())
		if packageName == "" {
			packageName = string(f.GoPackageName)
		}

		err = temp.ExecuteTemplate(genFile, "base.go.txt", map[string]interface{}{
			"pluginVersion": version.Version,
			"sourcePath":    f.Desc.Path(),
			"package":       packageName,
			"imports":       requiredImports,
			"enums":         f.Desc.Enums(),
			"messages":      f.Desc.Messages(),
			"services":      f.Desc.Services(),
		})
		if err != nil {
			return nil, err
		}
	}

	return plugin.Response(), nil
}

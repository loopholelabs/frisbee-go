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
	interfaceOpen         = " interface {"
	functionNameOpen      = "(context.Context, "
	functionNameSeparator = ") ("
	functionNameClose     = ", error)"
)

func writeInterface(f File, name string, methods protoreflect.MethodDescriptors) {
	builder := new(strings.Builder)
	f.P(typeOpen, utils.CamelCase(name), interfaceOpen)
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		builder.WriteString(pointer)
		builder.WriteString(utils.CamelCase(string(method.Input().FullName())))

		inputType := builder.String()
		builder.Reset()

		builder.WriteString(pointer)
		builder.WriteString(utils.CamelCase(string(method.Output().FullName())))

		outputType := builder.String()
		builder.Reset()

		f.P(tab, utils.CamelCase(string(method.Name())), functionNameOpen, inputType, functionNameSeparator, outputType, functionNameClose)
	}
	f.P(typeClose)
	f.P()
}

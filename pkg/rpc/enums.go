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
	enumType           = " uint32"
	enumDefinitionOpen = "const ("
)

func writeEnums(f File, name string, enums protoreflect.EnumValueDescriptors) {
	enumName := utils.CamelCase(name)
	f.P(typeOpen, enumName, enumType)
	f.P()
	f.P(enumDefinitionOpen)
	for i := 0; i < enums.Len(); i++ {
		enum := enums.Get(i)
		f.P(tab, utils.CamelCase(string(enum.FullName())), space, equal, space, enumName, parenthesesOpen, i, parenthesesClose)
	}
	f.P(parenthesesClose)
	f.P()
}

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

const (
	importOpenHeader = "import ("
)

var (
	requiredImports = []string{
		"github.com/loopholelabs/frisbee",
		"github.com/loopholelabs/packet",
		"github.com/loopholelabs/packet/pkg/packer",
		"github.com/rs/zerolog",
		"crypto/tls",
		"github.com/pkg/errors",
		"context",
		"sync",
		"sync/atomic",
	}
)

func writeImports(f File, imports []string) {
	f.P()
	f.P(importOpenHeader)
	for _, im := range imports {
		f.P("\t\"", im, "\"")
	}
	f.P(parenthesesClose)
	f.P()
}

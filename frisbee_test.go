/*
	Copyright 2021 Loophole Labs

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

package frisbee_test

import (
	"github.com/loophole-labs/frisbee"
	"github.com/rs/zerolog"
	"os"
)

func ExampleNewClient() {
	router := make(frisbee.ClientRouter)

	router[0] = func(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	frisbee.NewClient("127.0.0.1:8080", router, frisbee.WithLogger(&logger))
}

func ExampleNewServer() {
	router := make(frisbee.ServerRouter)

	router[0] = func(c *frisbee.Async, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	frisbee.NewServer("127.0.0.1:8080", router, frisbee.WithLogger(&logger))
}

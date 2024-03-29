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

package frisbee_test

import (
	"context"
	"github.com/loopholelabs/frisbee-go"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"github.com/rs/zerolog"
	"os"
)

func ExampleNewClient() {
	handlerTable := make(frisbee.HandlerTable)

	handlerTable[10] = func(ctx context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	_, _ = frisbee.NewClient(handlerTable, context.Background(), frisbee.WithLogger(&logger))
}

func ExampleNewServer() {
	handlerTable := make(frisbee.HandlerTable)

	handlerTable[10] = func(ctx context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action frisbee.Action) {
		return
	}

	logger := zerolog.New(os.Stdout)

	_, _ = frisbee.NewServer(handlerTable, frisbee.WithLogger(&logger))
}

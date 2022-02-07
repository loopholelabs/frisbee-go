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

package frisbee

import (
	"context"
	"github.com/loopholelabs/frisbee/internal/protocol"
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/pkg/errors"
)

// These are various frisbee errors that can be returned by the client or server:
var (
	InvalidContentLength     = errors.New("invalid content length")
	ConnectionClosed         = errors.New("connection closed")
	ConnectionNotInitialized = errors.New("connection not initialized")
	InvalidBufferLength      = errors.New("invalid buffer length")
	InvalidHandlerTable      = errors.New("invalid handler table configuration, a reserved value may have been used")
)

// Action is an ENUM used to modify the state of the client or server from a Handler function
//
//	NONE: used to do nothing (default)
//	CLOSE: close the frisbee connection
//	SHUTDOWN: shutdown the frisbee client or server
type Action int

// These are various frisbee actions, used to modify the state of the client or server from a Handler function:
const (
	// NONE is used to do nothing (default)
	NONE = Action(iota)

	// CLOSE is used to close the frisbee connection
	CLOSE

	// SHUTDOWN is used to shutdown the frisbee client or server
	SHUTDOWN
)

// Handler is the handler function called by frisbee for incoming packets of data, depending on the packet's Metadata.Operation field
type Handler func(ctx context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action)

// HandlerTable is the lookup table for Frisbee handler functions - based on the Metadata.Operation field of a packet,
// Frisbee will look up the correct handler for that packet.
type HandlerTable map[uint16]Handler

// These are internal reserved message types, and are the reason you cannot use 0-9 in Handler functions:
const (
	// HEARTBEAT is used to send heartbeats from the client to the server (and measure round trip time)
	HEARTBEAT = uint16(iota)

	// PING is used to check if a client is still alive
	PING

	// PONG is used to respond to a PING message
	PONG

	RESERVED3
	RESERVED4
	RESERVED5
	RESERVED6
	RESERVED7
	RESERVED8
	RESERVED9
)

var (
	// HEARTBEATPacket is a pre-allocated Frisbee Packet for HEARTBEAT Messages
	HEARTBEATPacket = &packet.Packet{
		Metadata: &protocol.Message{
			Operation: HEARTBEAT,
		},
	}

	// PINGPacket is a pre-allocated Frisbee Packet for PING Messages
	PINGPacket = &packet.Packet{
		Metadata: &protocol.Message{
			Operation: PING,
		},
	}

	// PONGPacket is a pre-allocated Frisbee Packet for PONG Messages
	PONGPacket = &packet.Packet{
		Metadata: &protocol.Message{
			Operation: PONG,
		},
	}
)

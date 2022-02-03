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
	InvalidRouter            = errors.New("invalid router configuration, a reserved value may have been used")
)

// Action is an ENUM used to modify the state of the client or server from a router function
//
//	NONE: used to do nothing (default)
//	CLOSE: close the frisbee connection
//	SHUTDOWN: shutdown the frisbee client or server
type Action int

// These are various frisbee actions, used to modify the state of the client or server from a router function:
const (
	// NONE is used to do nothing (default)
	NONE = Action(iota)

	// CLOSE is used to close the frisbee connection
	CLOSE

	// SHUTDOWN is used to shutdown the frisbee client or server
	SHUTDOWN
)

// These are internal reserved message types, and are the reason you cannot use 0-9 in the ClientRouter or the ServerRouter:
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
		Message: &protocol.Message{
			Operation: HEARTBEAT,
		},
	}

	// PINGPacket is a pre-allocated Frisbee Packet for PING Messages
	PINGPacket = &packet.Packet{
		Message: &protocol.Message{
			Operation: PING,
		},
	}

	// PONGPacket is a pre-allocated Frisbee Packet for PONG Messages
	PONGPacket = &packet.Packet{
		Message: &protocol.Message{
			Operation: PONG,
		},
	}
)

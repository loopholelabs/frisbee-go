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
	"github.com/loophole-labs/frisbee/internal/errors"
	"github.com/loophole-labs/frisbee/internal/protocol"
)

// These are various frisbee error contexts that can be returned by the client or server:
const (
	// DIAL is the error returned by the frisbee connection if dialing the server fails
	DIAL errors.ErrorContext = "error while dialing connection"

	// WRITECONN is the error returned by the frisbee client or server if writing to a frisbee connection fails
	WRITECONN errors.ErrorContext = "error while writing to frisbee connection"

	// READCONN is the error returned by the frisbee client or server if reading from a frisbee connection fails
	READCONN errors.ErrorContext = "error while reading from frisbee connection"

	// WRITE is the error returned by a frisbee connection if a write to the underlying TCP connection fails
	WRITE errors.ErrorContext = "error while writing to buffer"

	// PUSH is the error returned by a frisbee connection if a push to the message queue fails
	PUSH errors.ErrorContext = "error while pushing packet to message queue"

	// POP is the error returned by a frisbee connection if a pop from the message queue fails
	POP errors.ErrorContext = "error while popping packet from message queue"

	// ACCEPT is the error returned by a frisbee server if the underlying TCP server is unable to accept a new connection
	ACCEPT errors.ErrorContext = "error while accepting connections"
)

// These are various frisbee errors that can be returned by the client or server:
var (
	InvalidContentLength     = errors.New("invalid content length")
	ConnectionClosed         = errors.New("connection closed")
	ConnectionPaused         = errors.New("connection paused")
	ConnectionNotInitialized = errors.New("connection not initialized")
	InvalidBufferContents    = errors.New("invalid buffer contents")
	InvalidBufferLength      = errors.New("invalid buffer length")
)

// Action is an ENUM used to modify the state of the client or server from a router function
//
//	NONE: used to do nothing (default)
//	CLOSE: close the frisbee connection
//	SHUTDOWN: shutdown the frisbee client or server
type Action int

// Message is the structured frisbee message, and contains the following:
//
//	type MessageV0 struct {
//		From          uint32 // 4 Bytes
//		To            uint32 // 4 Bytes
//		Id            uint32 // 4 Bytes
//		Operation     uint32 // 4 Bytes
//		ContentLength uint64 // 8 Bytes
//	}
//
// These fields can be used however the user sees fit, however ContentLength must match the length of the content being
// delivered with the frisbee message (see the Conn.WriteMessage function for more details).
type Message protocol.Message

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
	HEARTBEAT = uint32(iota)
	STREAM
	STREAMCLOSE
	RESERVED4
	RESERVED5
	RESERVED6
	RESERVED7
	RESERVED8
	RESERVED9
)

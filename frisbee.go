// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"context"
	"errors"
	"time"

	"github.com/loopholelabs/polyglot/v2"

	"github.com/loopholelabs/frisbee-go/pkg/metadata"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
)

// These are various frisbee errors that can be returned by the client or server:
var (
	InvalidContentLength     = errors.New("invalid content length")
	ConnectionClosed         = errors.New("connection closed")
	StreamClosed             = errors.New("stream closed")
	InvalidStreamPacket      = errors.New("invalid stream packet")
	ConnectionNotInitialized = errors.New("connection not initialized")
	InvalidMagicHeader       = errors.New("invalid magic header")
	InvalidBufferLength      = errors.New("invalid buffer length")
	InvalidHandlerTable      = errors.New("invalid handler table configuration, a reserved value may have been used")
	InvalidOperation         = errors.New("invalid operation in packet, a reserved value may have been used")
	ContentLengthExceeded    = errors.New("content length exceeds maximum allowed")
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
)

// Handler is the handler function called by frisbee for incoming packets of data, depending on the packet's Metadata.Operation field
type Handler func(ctx context.Context, incoming *packet.Packet) (outgoing *packet.Packet, action Action)

// HandlerTable is the lookup table for Frisbee handler functions - based on the Metadata.Operation field of a packet,
// Frisbee will look up the correct handler for that packet.
type HandlerTable map[uint16]Handler

// These are internal reserved packet types, and are the reason you cannot use 0-9 in Handler functions:
const (
	// PING is used to check if a client is still alive
	PING = uint16(iota)

	// PONG is used to respond to a PING packets
	PONG

	// STREAM is used to request that a new stream be created by the receiver to
	// receive packets with the same packet ID until a packet with a ContentLength of 0 is received
	STREAM

	RESERVED3
	RESERVED4
	RESERVED5
	RESERVED6
	RESERVED7
	RESERVED8
	RESERVED9
)

var (
	// PINGPacket is a pre-allocated Frisbee Packet for PING Packets
	PINGPacket = &packet.Packet{
		Metadata: &metadata.Metadata{
			Operation: PING,
		},
		Content: polyglot.NewBuffer(),
	}

	// PONGPacket is a pre-allocated Frisbee Packet for PONG Packets
	PONGPacket = &packet.Packet{
		Metadata: &metadata.Metadata{
			Operation: PONG,
		},
		Content: polyglot.NewBuffer(),
	}
)

// temporary is an interface used to check if an error is recoverable
type temporary interface {
	Temporary() bool
}

const (
	// maxBackoff is the maximum amount ot time to wait before retrying to accept from a listener
	maxBackoff = time.Second

	// minBackoff is the minimum amount ot time to wait before retrying to accept from a listener
	minBackoff = time.Millisecond * 5
)

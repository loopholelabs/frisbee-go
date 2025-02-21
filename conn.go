// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"time"

	"github.com/loopholelabs/logging/types"

	"github.com/loopholelabs/frisbee-go/pkg/packet"
)

// DefaultBufferSize is the size of the default buffer
const DefaultBufferSize = 1 << 16

var (
	DefaultDeadline         = time.Second * 5
	DefaultPingInterval     = time.Millisecond * 500
	DefaultMaxContentLength = uint32(5 * 1024 * 1024) // 5 MB

	emptyTime = time.Time{}
	pastTime  = time.Unix(1, 0)

	emptyState = tls.ConnectionState{}
)

var (
	NotTLSConnectionError = errors.New("connection is not of type *tls.Conn")
)

type Conn interface {
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	ConnectionState() (tls.ConnectionState, error)
	Handshake() error
	HandshakeContext(context.Context) error
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	WritePacket(*packet.Packet) error
	ReadPacket() (*packet.Packet, error)
	Logger() types.Logger
	Error() error
	Raw() net.Conn
}

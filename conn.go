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

package frisbee

import (
	"context"
	"crypto/tls"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"io"
	"net"
	"time"
)

// DefaultBufferSize is the size of the default buffer
const DefaultBufferSize = 1 << 16

var (
	defaultLogger = zerolog.New(io.Discard)

	DefaultDeadline     = time.Second * 5
	DefaultPingInterval = time.Millisecond * 500

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
	Logger() *zerolog.Logger
	Error() error
	Raw() net.Conn
}

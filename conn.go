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
	"crypto/tls"
	"github.com/loopholelabs/frisbee/pkg/packet"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"net"
	"os"
	"time"
)

// DefaultBufferSize is the size of the default buffer
const DefaultBufferSize = 1 << 19

var (
	defaultLogger   = zerolog.New(os.Stdout)
	defaultDeadline = time.Second
	emptyTime       = time.Time{}
	pastTime        = time.Unix(1, 0)
)

var (
	NotTLSConnectionError = errors.New("connection is not of type *tls.Conn")
)

type Conn interface {
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	ConnectionState() (tls.ConnectionState, error)
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	WriteMessage(*packet.Packet) error
	ReadMessage() (*packet.Packet, error)
	Logger() *zerolog.Logger
	Error() error
	Raw() net.Conn
}

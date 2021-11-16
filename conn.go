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
	"bytes"
	"crypto/tls"
	"github.com/loopholelabs/frisbee/internal/errors"
	"github.com/rs/zerolog"
	"net"
	"os"
	"sync"
	"time"
)

// DefaultBufferSize is the size of the default buffer
const DefaultBufferSize = 1 << 19

// These are states that frisbee connections can be in:
const (
	// CONNECTED is used to specify that the connection is functioning normally
	CONNECTED = int32(iota)

	// CLOSED is used to specify that the connection has been closed (possibly due to an error)
	CLOSED
)

var (
	defaultLogger   = zerolog.New(os.Stdout)
	defaultDeadline = time.Second
	emptyTime       = time.Time{}
)

var (
	NotTLSConnectionError = errors.New("connection is not of type *tls.Conn")
)

type incomingBuffer struct {
	sync.Mutex
	buffer *bytes.Buffer
}

func newIncomingBuffer() *incomingBuffer {
	return &incomingBuffer{
		buffer: bytes.NewBuffer(make([]byte, 0, DefaultBufferSize)),
	}
}

type Conn interface {
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	ConnectionState() (tls.ConnectionState, error)
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	WriteMessage(*Message, *[]byte) error
	ReadMessage() (*Message, *[]byte, error)
	Logger() *zerolog.Logger
	Error() error
	Raw() net.Conn
}

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
	"crypto/rand"
	"github.com/loopholelabs/frisbee-go/pkg/packet"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"testing"
	"time"
)

func TestNewStream(t *testing.T) {
	t.Parallel()

	const packetSize = 512

	emptyLogger := zerolog.New(io.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	writerStream := writerConn.Stream(0)

	data := make([]byte, packetSize)
	_, err := rand.Read(data)
	require.NoError(t, err)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Metadata.ContentLength = uint32(packetSize)
	p.Content.Write(data)

	err = writerStream.WritePacket(p)
	require.NoError(t, err)
	packet.Put(p)

	var readerStream *Stream
	timer := time.NewTimer(DefaultDeadline)
	select {
	case <-timer.C:
		t.Fatal("timed out waiting for reader stream")
	case readerStream = <-readerConn.StreamCh():
	}

	p, err = readerStream.ReadPacket()
	require.NoError(t, err)
	require.NotNil(t, p.Metadata)
	assert.Equal(t, readerStream.ID(), p.Metadata.Id)
	assert.Equal(t, STREAM, p.Metadata.Operation)
	assert.Equal(t, uint32(packetSize), p.Metadata.ContentLength)
	assert.Equal(t, data, p.Content.Bytes())

	err = readerStream.Close()
	require.NoError(t, err)

	time.Sleep(DefaultDeadline)

	err = writerStream.Close()
	require.ErrorIs(t, err, StreamClosed)

	err = readerConn.Close()
	assert.NoError(t, err)
	err = writerConn.Close()
	assert.NoError(t, err)
}

func TestNewStreamStale(t *testing.T) {
	t.Parallel()

	const packetSize = 512

	emptyLogger := zerolog.New(io.Discard)

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

	writerStream := writerConn.Stream(0)

	data := make([]byte, packetSize)
	_, err := rand.Read(data)
	require.NoError(t, err)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Metadata.ContentLength = uint32(packetSize)
	p.Content.Write(data)

	err = writerStream.WritePacket(p)
	require.NoError(t, err)

	err = writerStream.WritePacket(p)
	require.NoError(t, err)

	packet.Put(p)

	var readerStream *Stream
	timer := time.NewTimer(DefaultDeadline)
	select {
	case <-timer.C:
		t.Fatal("timed out waiting for reader stream")
	case readerStream = <-readerConn.StreamCh():
	}

	err = writerStream.Close()
	require.NoError(t, err)

	time.Sleep(DefaultDeadline)

	p, err = readerStream.ReadPacket()
	require.NoError(t, err)
	require.NotNil(t, p.Metadata)
	assert.Equal(t, readerStream.ID(), p.Metadata.Id)
	assert.Equal(t, STREAM, p.Metadata.Operation)
	assert.Equal(t, uint32(packetSize), p.Metadata.ContentLength)
	assert.Equal(t, data, p.Content.Bytes())

	p, err = readerStream.ReadPacket()
	require.NoError(t, err)
	require.NotNil(t, p.Metadata)
	assert.Equal(t, readerStream.ID(), p.Metadata.Id)
	assert.Equal(t, STREAM, p.Metadata.Operation)
	assert.Equal(t, uint32(packetSize), p.Metadata.ContentLength)
	assert.Equal(t, data, p.Content.Bytes())

	_, err = readerStream.ReadPacket()
	require.ErrorIs(t, err, StreamClosed)

	err = readerConn.Close()
	assert.NoError(t, err)
}

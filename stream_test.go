// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"crypto/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/loopholelabs/logging"

	"github.com/loopholelabs/frisbee-go/pkg/packet"
)

func TestNewStream(t *testing.T) {
	t.Parallel()

	const packetSize = 512

	emptyLogger := logging.Test(t, logging.Noop, t.Name())
	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	writerStream := writerConn.NewStream(0)

	data := make([]byte, packetSize)
	_, err := rand.Read(data)
	require.NoError(t, err)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Metadata.ContentLength = uint32(packetSize)
	p.Content.Write(data)

	readerStreamCh := make(chan *Stream)
	var readerStream *Stream

	readerConn.SetNewStreamHandler(func(stream *Stream) {
		readerStreamCh <- stream
	})

	err = writerStream.WritePacket(p)
	require.NoError(t, err)
	packet.Put(p)

	timer := time.NewTimer(DefaultDeadline)
	select {
	case <-timer.C:
		t.Fatal("timed out waiting for reader stream")
	case readerStream = <-readerStreamCh:
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

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, emptyLogger)
	writerConn := NewAsync(writer, emptyLogger)

	writerStream := writerConn.NewStream(0)

	data := make([]byte, packetSize)
	_, err := rand.Read(data)
	require.NoError(t, err)

	p := packet.Get()
	p.Metadata.Id = 64
	p.Metadata.Operation = 32
	p.Metadata.ContentLength = uint32(packetSize)
	p.Content.Write(data)

	readerStreamCh := make(chan *Stream)
	var readerStream *Stream

	readerConn.SetNewStreamHandler(func(stream *Stream) {
		readerStreamCh <- stream
	})

	err = writerStream.WritePacket(p)
	require.NoError(t, err)

	err = writerStream.WritePacket(p)
	require.NoError(t, err)

	packet.Put(p)

	timer := time.NewTimer(DefaultDeadline)
	select {
	case <-timer.C:
		t.Fatal("timed out waiting for reader stream")
	case readerStream = <-readerStreamCh:
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

func TestNewStreamDualCreate(t *testing.T) {
	t.Parallel()

	const packetSize = 512

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, emptyLogger, func(_ *Stream) {})
	writerConn := NewAsync(writer, emptyLogger, func(_ *Stream) {})

	writerStream := writerConn.NewStream(0)
	readerStream := readerConn.NewStream(0)

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

func TestStreamConnClose(t *testing.T) {
	t.Parallel()

	const packetSize = 512

	emptyLogger := logging.Test(t, logging.Noop, t.Name())

	reader, writer := net.Pipe()

	readerConn := NewAsync(reader, emptyLogger, func(_ *Stream) {})
	writerConn := NewAsync(writer, emptyLogger, func(_ *Stream) {})

	writerStream := writerConn.NewStream(0)
	readerStream := readerConn.NewStream(0)

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

	err = writerConn.Close()
	require.NoError(t, err)

	time.Sleep(DefaultDeadline)

	err = writerStream.Close()
	require.ErrorIs(t, err, StreamClosed)

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

//go:build !race
// +build !race

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
	"bufio"
	"github.com/loopholelabs/testing/conn/pair"
	"github.com/rs/zerolog"
	"io"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

func BenchmarkAsyncThroughputLarge(b *testing.B) {
	const testSize = 100

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	readerConn := NewAsync(reader, &emptyLogger, false)
	writerConn := NewAsync(writer, &emptyLogger, false)

	b.Run("1MB", throughputRunner(testSize, 1<<20, readerConn, writerConn))
	b.Run("2MB", throughputRunner(testSize, 1<<21, readerConn, writerConn))
	b.Run("4MB", throughputRunner(testSize, 1<<22, readerConn, writerConn))
	b.Run("8MB", throughputRunner(testSize, 1<<23, readerConn, writerConn))
	b.Run("16MB", throughputRunner(testSize, 1<<24, readerConn, writerConn))

	_ = readerConn.Close()
	_ = writerConn.Close()
}

func BenchmarkSyncThroughputLarge(b *testing.B) {
	const testSize = 100

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	readerConn := NewSync(reader, &emptyLogger)
	writerConn := NewSync(writer, &emptyLogger)

	b.Run("1MB", throughputRunner(testSize, 1<<20, readerConn, writerConn))
	b.Run("2MB", throughputRunner(testSize, 1<<21, readerConn, writerConn))
	b.Run("4MB", throughputRunner(testSize, 1<<22, readerConn, writerConn))
	b.Run("8MB", throughputRunner(testSize, 1<<23, readerConn, writerConn))
	b.Run("16MB", throughputRunner(testSize, 1<<24, readerConn, writerConn))

	_ = readerConn.Close()
	_ = writerConn.Close()
}

func BenchmarkTCPThroughput(b *testing.B) {
	const testSize = 100

	reader, writer, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	TCPThroughputRunner := func(testSize uint32, packetSize uint32, readerConn net.Conn, writerConn net.Conn) func(*testing.B) {
		bufWriter := bufio.NewWriter(writerConn)
		bufReader := bufio.NewReader(readerConn)
		return func(b *testing.B) {
			b.SetBytes(int64(testSize * packetSize))
			b.ReportAllocs()
			var err error

			randomData := make([]byte, packetSize)
			readData := make([]byte, packetSize)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				done := make(chan struct{}, 1)
				errCh := make(chan error, 1)
				go func() {
					for i := uint32(0); i < testSize; i++ {
						err := readerConn.SetReadDeadline(time.Now().Add(defaultDeadline))
						if err != nil {
							errCh <- err
							return
						}
						_, err = io.ReadAtLeast(bufReader, readData[0:], int(packetSize))
						if err != nil {
							errCh <- err
							return
						}
					}
					done <- struct{}{}
				}()
				for i := uint32(0); i < testSize; i++ {
					select {
					case err = <-errCh:
						b.Fatal(err)
					default:
						err = writerConn.SetWriteDeadline(time.Now().Add(defaultDeadline))
						if err != nil {
							b.Fatal(err)
						}
						_, err = bufWriter.Write(randomData)
						if err != nil {
							b.Fatal(err)
						}
					}
				}
				err = writerConn.SetWriteDeadline(time.Now().Add(defaultDeadline))
				if err != nil {
					b.Fatal(err)
				}
				err = bufWriter.Flush()
				if err != nil {
					b.Fatal(err)
				}
				select {
				case <-done:
					continue
				case err := <-errCh:
					b.Fatal(err)
				}
			}
			b.StopTimer()
		}
	}

	b.Run("32 Bytes", TCPThroughputRunner(testSize, 32, reader, writer))
	b.Run("512 Bytes", TCPThroughputRunner(testSize, 512, reader, writer))
	b.Run("1024 Bytes", TCPThroughputRunner(testSize, 1024, reader, writer))
	b.Run("2048 Bytes", TCPThroughputRunner(testSize, 2048, reader, writer))
	b.Run("4096 Bytes", TCPThroughputRunner(testSize, 4096, reader, writer))
	b.Run("1MB", TCPThroughputRunner(testSize, 1<<20, reader, writer))
	b.Run("2MB", TCPThroughputRunner(testSize, 1<<21, reader, writer))
	b.Run("4MB", TCPThroughputRunner(testSize, 1<<22, reader, writer))
	b.Run("8MB", TCPThroughputRunner(testSize, 1<<23, reader, writer))
	b.Run("16MB", TCPThroughputRunner(testSize, 1<<24, reader, writer))

	_ = reader.Close()
	_ = writer.Close()
}

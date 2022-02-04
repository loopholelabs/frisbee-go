//go:build !race
// +build !race

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
	"github.com/loopholelabs/testing/conn/pair"
	"github.com/rs/zerolog"
	"io/ioutil"
	"testing"
)

func BenchmarkAsyncThroughputLarge(b *testing.B) {
	const testSize = 100

	emptyLogger := zerolog.New(ioutil.Discard)

	reader, writer, err := pair.New()
	if err != nil {
		b.Fatal(err)
	}

	readerConn := NewAsync(reader, &emptyLogger)
	writerConn := NewAsync(writer, &emptyLogger)

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

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
	"github.com/loopholelabs/frisbee/pkg/packet"
	"go.uber.org/goleak"
	"testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func throughputRunner(testSize uint32, packetSize uint32, readerConn Conn, writerConn Conn) func(b *testing.B) {
	return func(b *testing.B) {
		b.SetBytes(int64(testSize * packetSize))
		b.ReportAllocs()
		var err error

		randomData := make([]byte, packetSize)

		p := packet.Get()
		p.Metadata.Id = 64
		p.Metadata.Operation = 32
		p.Content.Write(randomData)
		p.Metadata.ContentLength = packetSize
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			done := make(chan struct{}, 1)
			errCh := make(chan error, 1)
			go func() {
				for i := uint32(0); i < testSize; i++ {
					p, err := readerConn.ReadPacket()
					if err != nil {
						errCh <- err
						return
					}
					packet.Put(p)
				}
				done <- struct{}{}
			}()
			for i := uint32(0); i < testSize; i++ {
				select {
				case err = <-errCh:
					b.Fatal(err)
				default:
					err = writerConn.WritePacket(p)
					if err != nil {
						b.Fatal(err)
					}
				}
			}
			select {
			case <-done:
				continue
			case err = <-errCh:
				b.Fatal(err)
			}
		}
		b.StopTimer()

		packet.Put(p)
	}
}

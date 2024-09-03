// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/loopholelabs/frisbee-go/pkg/packet"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func throughputRunner(testSize, packetSize uint32, readerConn, writerConn Conn) func(b *testing.B) {
	return func(b *testing.B) {
		b.SetBytes(int64(testSize * packetSize))
		b.ReportAllocs()

		randomData := make([]byte, packetSize)
		_, err := rand.Read(randomData)
		require.NoError(b, err)

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

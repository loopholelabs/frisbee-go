// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"github.com/loopholelabs/common/pkg/pool"
)

type Buffer [Size]byte

func NewBuffer() *Buffer {
	return new(Buffer)
}

func (b *Buffer) Reset() {}

var (
	bufferPool = pool.NewPool[Buffer, *Buffer](NewBuffer)
)

func GetBuffer() *Buffer {
	return bufferPool.Get()
}

func PutBuffer(b *Buffer) {
	bufferPool.Put(b)
}

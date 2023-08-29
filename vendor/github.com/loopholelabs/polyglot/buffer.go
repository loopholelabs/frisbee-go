/*
	Copyright 2023 Loophole Labs

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

package polyglot

const (
	defaultSize = 512
)

type Buffer []byte

func (buf *Buffer) Reset() {
	*buf = (*buf)[:0]
}

func (buf *Buffer) Write(b []byte) int {
	if cap(*buf)-len(*buf) < len(b) {
		*buf = append((*buf)[:len(*buf)], b...)
	} else {
		*buf = (*buf)[:len(*buf)+copy((*buf)[len(*buf):cap(*buf)], b)]
	}
	return len(b)
}

func NewBuffer() *Buffer {
	c := make(Buffer, 0, defaultSize)
	return &c
}

func (buf *Buffer) Bytes() []byte {
	return *buf
}

func (buf *Buffer) Len() int {
	return len(*buf)
}

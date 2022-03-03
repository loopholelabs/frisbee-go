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

package packet

const (
	defaultSize = 512
)

// Content is a packet's byte buffer that grows and is recycled as required
type Content struct {
	B []byte
}

// NewContent returns a new Content struct with an allocated array of size defaultSize
func NewContent() *Content {
	return &Content{
		B: make([]byte, 0, defaultSize),
	}
}

// Reset resets the underlying byte slice for future use
func (c *Content) Reset() {
	c.B = c.B[:0]
}

// Write efficiently copies the byte slice b into the content buffer, however it
// does *not* update the content length.
func (c *Content) Write(b []byte) int {
	if cap(c.B)-len(c.B) < len(b) {
		c.B = append(c.B[:len(c.B)], b...)
	} else {
		c.B = c.B[:len(c.B)+copy(c.B[len(c.B):cap(c.B)], b)]
	}
	return len(b)
}

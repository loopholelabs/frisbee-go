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

package queue

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRound(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		in       uint64
		expected uint64
	}{
		{in: 0, expected: 0x0},
		{in: 1, expected: 0x1},
		{in: 2, expected: 0x2},
		{in: 3, expected: 0x4},
		{in: 4, expected: 0x4},
		{in: 5, expected: 0x8},
		{in: 7, expected: 0x8},
		{in: 8, expected: 0x8},
		{in: 9, expected: 0x10},
		{in: 16, expected: 0x10},
		{in: 32, expected: 0x20},
		{in: 0xFFFFFFF0, expected: 0x100000000},
		{in: 0xFFFFFFFF, expected: 0x100000000},
	}
	for _, tc := range tcs {
		assert.Equalf(t, tc.expected, round(tc.in), "in: %d", tc.in)
	}
}

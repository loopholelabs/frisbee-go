package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCamelCase(t *testing.T) {
	tcs := []struct {
		caseName string
		in       string
		expected string
	}{
		{
			caseName: "simple",
			in:       "test",
			expected: "Test",
		},
		{
			caseName: "empty",
			in:       "",
			expected: "",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.caseName, func(t *testing.T) {
			assert.Equal(t, tc.expected, CamelCase(tc.in))
		})
	}
}

package frisbee

import "testing"

func DisableMaxContentLength(tb testing.TB) uint32 {
	tb.Helper()

	oldMax := DefaultMaxContentLength
	DefaultMaxContentLength = 0
	tb.Cleanup(func() { DefaultMaxContentLength = oldMax })

	return oldMax
}

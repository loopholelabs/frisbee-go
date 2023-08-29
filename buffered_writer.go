package frisbee

import (
	"bufio"
	"io"
	"sync"
)

type BufferedWriter struct {
	w *bufio.Writer

	mu sync.RWMutex
}

func NewBufferedWriterSize(w io.Writer, size int) *BufferedWriter {
	return &BufferedWriter{
		w: bufio.NewWriterSize(w, size),
	}
}
func (bw *BufferedWriter) Buffered() int {
	bw.mu.RLock()
	defer bw.mu.RUnlock()
	return bw.w.Buffered()
}
func (bw *BufferedWriter) Write(p []byte) (int, error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.w.Write(p)
}
func (bw *BufferedWriter) Flush() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	return bw.w.Flush()
}

package frisbee

import "sync"

type Streams struct {
	mu sync.RWMutex
	ms map[uint16]*Stream
	hs *StreamHandlers
}

func NewStreams() *Streams {
	return &Streams{
		ms: make(map[uint16]*Stream),
		hs: NewStreamHandlers(1, 1),
	}
}

func (s *Streams) Remove(id uint16) {
	s.mu.Lock()
	delete(s.ms, id)
	s.mu.Unlock()
}

func (s *Streams) Get(id uint16) *Stream {
	s.mu.RLock()
	stream := s.ms[id]
	s.mu.RUnlock()
	return stream
}

func (s *Streams) CreateWithCheckOfExistence(id uint16, f func() *Stream) *Stream {
	s.mu.Lock()
	var stream *Stream
	if stream = s.ms[id]; stream == nil {
		stream = f()
		s.ms[id] = stream
	}
	s.mu.Unlock()

	return stream
}

func (s *Streams) Create(id uint16, f func() *Stream) *Stream {
	s.mu.Lock()

	stream := f()
	s.ms[id] = stream
	s.mu.Unlock()

	return stream
}

func (s *Streams) Handlers() *StreamHandlers {
	return s.hs
}

func (s *Streams) CloseAll() {
	s.mu.RLock()
	for _, stream := range s.ms {
		_ = stream.Close()
	}
	s.mu.RUnlock()
}

type StreamHandler func(*Stream)

type StreamHandlers struct {
	mu sync.RWMutex
	f  []StreamHandler
}

func NewStreamHandlers(cap, len int) *StreamHandlers {
	return &StreamHandlers{
		f: make([]StreamHandler, cap, len),
	}
}

func (sh *StreamHandlers) Set(handler StreamHandler) {
	sh.mu.Lock()
	sh.f[0] = handler
	sh.mu.Unlock()
}

func (sh *StreamHandlers) Get() StreamHandler {
	sh.mu.RLock()
	h := sh.f[0]
	sh.mu.RUnlock()

	return h
}

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

package frisbee

import "sync"

type Streams struct {
	mu sync.RWMutex
	ms map[uint16]*Stream
	hs *streamHandlers
}

// NewStreams do create wrapper around map of streams allowing to operate on
// multithreading environment
func NewStreams() *Streams {
	return &Streams{
		ms: make(map[uint16]*Stream),
		hs: newStreamHandlers(1, 1),
	}
}

// Remove an stream out of stream map by id
func (s *Streams) Remove(id uint16) {
	s.mu.Lock()
	delete(s.ms, id)
	s.mu.Unlock()
}

// Get an stream from map by id
func (s *Streams) Get(id uint16) *Stream {
	s.mu.RLock()
	stream := s.ms[id]
	s.mu.RUnlock()
	return stream
}

// CreateWithCheckOfExistence is creating a new stream if it doesn't exist,
// using the f parameter as a creation handler
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

// Create is creating a new stream by overwriting existing value if it already exists into the map,
// using the f parameter as a creation handler
func (s *Streams) Create(id uint16, f func() *Stream) *Stream {
	s.mu.Lock()

	stream := f()
	s.ms[id] = stream
	s.mu.Unlock()

	return stream
}

// Handler retrieve the stream handler
func (s *Streams) Handler() NewStreamHandler {
	return s.hs.Get()
}

// SetHandler set the stream handler
func (s *Streams) SetHandler(h NewStreamHandler) {
	s.hs.Set(h)
}

func (s *Streams) Close() {
	s.mu.RLock()
	for _, stream := range s.ms {
		_ = stream.Close()
	}
	s.mu.RUnlock()
}

// TODO: To be given a better name, like StreamHandler; Wasn't cgahnged as is used into the proto generated code
type NewStreamHandler func(*Stream)

type streamHandlers struct {
	mu sync.RWMutex
	f  []NewStreamHandler
}

func newStreamHandlers(cap, len int) *streamHandlers {
	return &streamHandlers{
		f: make([]NewStreamHandler, cap, len),
	}
}

func (sh *streamHandlers) Set(handler NewStreamHandler) {
	sh.mu.Lock()
	sh.f[0] = handler
	sh.mu.Unlock()
}

func (sh *streamHandlers) Get() NewStreamHandler {
	sh.mu.RLock()
	h := sh.f[0]
	sh.mu.RUnlock()

	return h
}

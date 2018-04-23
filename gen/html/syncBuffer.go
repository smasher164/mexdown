package html

import (
	"bytes"
	"io"
	"sync"
)

type syncBuffer struct {
	m   sync.RWMutex
	b   bytes.Buffer
	err error
}

func (s *syncBuffer) Read(p []byte) (n int, err error) {
	s.m.RLock()
	defer s.m.RUnlock()
	n, err = s.b.Read(p)
	if s.err != nil {
		err = s.err
	}
	return
}

func (s *syncBuffer) WriteTo(w io.Writer) (n int64, err error) {
	s.m.RLock()
	defer s.m.RUnlock()
	n, err = s.b.WriteTo(w)
	if s.err != nil {
		err = s.err
	}
	return
}

func (s *syncBuffer) Write(p []byte) (n int, err error) {
	s.m.Lock()
	defer s.m.Unlock()
	n, err = s.b.Write(p)
	return
}

func (s *syncBuffer) SetError(err error) {
	s.m.Lock()
	defer s.m.Unlock()
	s.err = err
}

func (s *syncBuffer) Reset() {
	s.m.Lock()
	defer s.m.Unlock()
	s.b.Reset()
	s.err = nil
}

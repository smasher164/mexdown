// MIT License

// Copyright (c) 2018 Akhil Indurti

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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

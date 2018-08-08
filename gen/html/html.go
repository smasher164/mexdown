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

// Package html converts an AST file structure into html output.
// Text inside code segments is automatically escaped.
// Overlapping format tags in the source are converted into a tree structure.
// Directives are parsed according to the Bourne shell's word-splitting rules.
//
// AST nodes correspond to the following HTML tags:
// 	Paragraph                   <p></p>
// 	Header                      <h1></h1>, <h2></h2>, <h3></h3>, <h4></h4>, <h5></h5>, <h6></h6>, <p></p>
// 	List                        <ul></ul>
// 	ListItem (bulleted)         <li class="bullet"></li>
// 	ListItem (labeled)          <li><span></span></li>
// 	Directive (raw string)      <pre></pre>
// 	Directive (with command)    Depends on the result of command execution
// 	Citation                    <a href=""></a>
// 	Italics                     <em></em>
// 	Bold                        <strong></strong>
// 	BoldItalic                  <strong><em></em></strong>
// 	Underline                   <u></u>
// 	Strikethrough               <s></s>
// 	Code Segment                <code></code>
package html // import "akhil.cc/mexdown/gen/html"

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"

	"akhil.cc/mexdown/ast"
	sq "github.com/kballard/go-shellquote"
)

type syncWriter struct {
	m sync.Mutex
	w io.Writer
}

func (s *syncWriter) Write(p []byte) (n int, err error) {
	s.m.Lock()
	defer s.m.Unlock()
	n, err = s.w.Write(p)
	return
}

type stickyCountWriter struct {
	n   int64
	err error
	w   io.Writer
}

func (c *stickyCountWriter) Write(p []byte) (n int, err error) {
	if c.err != nil {
		return 0, c.err
	}
	n, err = c.w.Write(p)
	c.err = err
	c.n += int64(n)
	return
}

// Generator represents a non-reusable HTML output generator for an *ast.File.
type Generator struct {
	// Stdout and Stderr specify the generator's standard output and standard error.
	//
	// HTML output will be written to standard out. Standard error is typically only
	// written by a process run for an *ast.Directive.
	//
	// If Stdout == Stderr, at most one goroutine at a time will call Write.
	Stdout   io.Writer
	Stderr   io.Writer
	ctx      context.Context
	file     *ast.File
	waitdone chan error

	m     sync.Mutex
	pipes []io.Closer
}

// Gen returns the Generator struct to convert the given file into HTML output.
//
// It sets only the file in the returned structure.
func Gen(file *ast.File) *Generator {
	return &Generator{ctx: context.TODO(), file: file}
}

// GenContext is like Gen but includes a context.
//
// The provided context is used both to halt HTML generation
// after processing an ast.Stmt, and to kill any processes executed
// for an *ast.Directive.
func GenContext(ctx context.Context, file *ast.File) *Generator {
	if ctx == nil {
		panic("nil context")
	}
	return &Generator{ctx: ctx, file: file}
}

// Start starts the generator but does not wait for it to complete.
func (g *Generator) Start() error {
	if g.Stdout == nil {
		g.Stdout = ioutil.Discard
	}
	if g.Stderr == nil {
		g.Stderr = ioutil.Discard
	}
	if g.Stdout == g.Stderr {
		g.Stdout = &syncWriter{w: g.Stdout}
		g.Stderr = g.Stdout
	}
	g.waitdone = make(chan error)
	go func() {
		err := g.gen()
		for _, p := range g.pipes {
			p.Close()
		}
		g.m.Lock()
		g.pipes = nil
		g.m.Unlock()
		g.waitdone <- err
	}()
	return nil
}

// Wait waits for the generator to complete and finish copying to
// Stdout and Stderr. It is an error to call Wait before Start
// has been called.
//
// Wait will release any resources associated with the generator.
func (g *Generator) Wait() error {
	if g.waitdone == nil {
		return fmt.Errorf("not started")
	}
	// prevent callers to Wait from a deadlock via not waiting for pipes to close
	g.m.Lock()
	if g.pipes != nil {
		g.m.Unlock()
		return fmt.Errorf("all reads from the pipe have not completed")
	}
	g.m.Unlock()
	err := <-g.waitdone
	close(g.waitdone)
	return err
}

// Run starts the generator and waits for it to complete, returning
// any errors enountered.
func (g *Generator) Run() error {
	if err := g.Start(); err != nil {
		return err
	}
	return g.Wait()
}

// StdoutPipe returns a pipe that is connected to the generator's
// standard output.
//
// It is invalid to call Wait until all reads from the pipe have completed.
// For the same reason, it is invalid to call Run when using StdoutPipe.
func (g *Generator) StdoutPipe() (io.Reader, error) {
	if g.Stdout != nil {
		return nil, fmt.Errorf("Stdout already set")
	}
	pr, pw := io.Pipe()
	g.Stdout = pw
	g.pipes = append(g.pipes, pw)
	return pr, nil
}

// StderrPipe returns a pipe that is connected to the generator's
// standard error.
//
// It is invalid to call Wait until all reads from the pipe have completed.
// For the same reason, it is invalid to call Run when using StderrPipe.
func (g *Generator) StderrPipe() (io.Reader, error) {
	if g.Stderr != nil {
		return nil, fmt.Errorf("Stderr already set")
	}
	pr, pw := io.Pipe()
	g.Stderr = pw
	g.pipes = append(g.pipes, pw)
	return pr, nil
}

// Output runs the generator and returns its standard output.
func (g *Generator) Output() ([]byte, error) {
	if g.Stdout != nil {
		return nil, fmt.Errorf("Stdout already set")
	}
	var stdout bytes.Buffer
	g.Stdout = &stdout
	err := g.Run()
	return stdout.Bytes(), err
}

// CombinedOutput runs the generator and returns its combined
// standard output and standard error.
func (g *Generator) CombinedOutput() ([]byte, error) {
	if g.Stdout != nil {
		return nil, fmt.Errorf("Stdout already set")
	}
	if g.Stderr != nil {
		return nil, fmt.Errorf("Stderr already set")
	}
	var b bytes.Buffer
	g.Stdout = &b
	g.Stderr = &b
	err := g.Run()
	return b.Bytes(), err
}

func (g *Generator) gen() error {
	cw := &stickyCountWriter{0, nil, g.Stdout}
	for i := range g.file.List {
		select {
		case <-g.ctx.Done():
			return cw.err
		default:
			switch t := g.file.List[i].(type) {
			case *ast.Paragraph:
				if len(t.Body) != 0 {
					cw.Write([]byte("<p>"))
					txt := ast.Text(*t)
					g.text(&txt, cw)
					cw.Write([]byte("</p>"))
				}
			case *ast.Header:
				var tag string
				if t.NThorpe > 6 {
					tag = "p"
				} else {
					tag = "h" + strconv.Itoa(t.NThorpe)
				}
				cw.Write([]byte("<" + tag + ">"))
				g.text(&t.Text, cw)
				cw.Write([]byte("</" + tag + ">"))
			case *ast.List:
				g.list(t, cw)
			case *ast.Directive:
				if len(t.Command) == 0 {
					fmt.Fprintf(cw, "<pre>%s</pre>", html.EscapeString(t.Raw))
				} else {
					words, err := sq.Split(t.Command)
					if err != nil {
						return err
					}
					if len(words) == 0 {
						return fmt.Errorf("No valid commands: '%q'", t.Command)
					}
					cmd := exec.CommandContext(g.ctx, words[0], words[1:]...)
					cmd.Stdin = strings.NewReader(t.Raw)
					cmd.Stdout = cw
					cmd.Stderr = g.Stderr
					if err := cmd.Run(); err != nil {
						return err
					}
				}
			}
		}
	}
	return cw.err
}

func replace(s, r string, pos, width int) string {
	return s[:pos] + r + s[pos+width:]
}

func open(tag int) bool {
	return tag&1 == 0
}

const (
	anchor = iota
	anchorClose
	italics
	italicsClose
	bold
	boldClose
	bolditalic
	bolditalicClose
	underline
	underlineClose
	strikethrough
	strikethroughClose
	code
	codeClose
)

var fstr = [...]string{
	anchor:               "<a href=%q>",
	anchorClose:          "</a>",
	italics:            "<em>",
	italicsClose:       "</em>",
	bold:               "<strong>",
	boldClose:          "</strong>",
	bolditalic:         "<strong><em>",
	bolditalicClose:    "</em></strong>",
	underline:          "<u>",
	underlineClose:     "</u>",
	strikethrough:      "<s>",
	strikethroughClose: "</s>",
	code:                "<code>",
	codeClose:           "</code>",
}

func (g *Generator) text(t *ast.Text, w io.Writer) (n int, err error) {
	type repl struct {
		i     int
		w     int
		kind  int
		extra int
	}
	rep := make([]repl, 0, 2*len(t.Format))
	insert := func(i int, r repl) []repl {
		rep = append(rep, repl{})
		copy(rep[i+1:], rep[i:])
		rep[i] = r
		return rep
	}
	offset := 0
	old := 0
	sort.Slice(t.Format, func(i, j int) bool { return t.Format[i].End < t.Format[j].End })
	for i, f := range t.Format {
		// escape code text
		if f.Kind == ast.Raw {
			beg := f.Beg + offset
			end := f.End + offset
			substr := t.Body[beg+1 : end]
			escRaw := html.EscapeString(substr)
			if len(escRaw) != len(substr) {
				old = f.End
			}
			t.Body = replace(t.Body, escRaw, beg+1, end-beg-1)
			offset += len(escRaw) - (end - beg - 1)
			t.Format[i].Beg = beg
			t.Format[i].End = beg + len(escRaw) + 1
		} else {
			if t.Format[i].Beg > old {
				t.Format[i].Beg += offset
			}
			if t.Format[i].End > old {
				t.Format[i].End += offset
			}
		}
	}
	for _, f := range t.Format {
		switch f.Kind {
		case ast.Cite:
			rep = append(rep, repl{f.Beg, f.End, anchor, 0})
			begClose := f.Beg
			if t.Body[f.End] == ')' {
				begClose += strings.Index(t.Body[f.Beg:], "](")
			}
			rep = append(rep, repl{f.End, f.End, anchorClose, begClose})
		case ast.Italic:
			rep = append(rep, repl{f.Beg, 1, italics, 0})
			rep = append(rep, repl{f.End, 1, italicsClose, 0})
		case ast.Bold:
			rep = append(rep, repl{f.Beg, 2, bold, 0})
			rep = append(rep, repl{f.End, 2, boldClose, 0})
		case ast.BoldItalic:
			rep = append(rep, repl{f.Beg, 3, bolditalic, 0})
			rep = append(rep, repl{f.End, 3, bolditalicClose, 0})
		case ast.Underline:
			rep = append(rep, repl{f.Beg, 1, underline, 0})
			rep = append(rep, repl{f.End, 1, underlineClose, 0})
		case ast.Strikethrough:
			rep = append(rep, repl{f.Beg, 2, strikethrough, 0})
			rep = append(rep, repl{f.End, 2, strikethroughClose, 0})
		case ast.Raw:
			rep = append(rep, repl{f.Beg, 1, code, 0})
			rep = append(rep, repl{f.End, 1, codeClose, 0})
		}
	}
	sort.Slice(rep, func(i, j int) bool { return rep[i].i < rep[j].i })
	// Fix format tree
	bottom := 0
	for current := 0; current < len(rep); current++ {
		// Find closing tag
		if !open(rep[current].kind) {
			// Walk backwards through list
			for lower := current - 1; lower >= bottom; lower-- {
				// Find first opening tag that does not match closing
				if rep[lower].kind != anchor && open(rep[lower].kind) && (rep[current].kind-rep[lower].kind) != 1 {
					rlower := rep[lower]
					rcurr := rep[current]
					// Insert its closing tag before our unmatched tag
					rep = insert(current, repl{rcurr.i, rcurr.w, rlower.kind + 1, rcurr.extra})
					rep[current+1].w = 0
					rep = insert(current+2, repl{rcurr.i, 0, rlower.kind, rcurr.extra})
				}
				current++
			}
			bottom = current + 1
		}
	}
	offset = 0
	for _, f := range rep {
		switch f.kind {
		case anchor:
			del := strings.Index(t.Body[f.i+offset:], "](") + f.i + offset
			src := t.Body[del+2 : f.w+offset]
			var citation string
			if t.Body[f.w+offset] != ')' {
				citation = fmt.Sprintf(fstr[f.kind], t.Body[f.i+offset+1:f.w+offset])
			} else if hSrc, ok := g.file.Cite[src]; ok {
				hSrc = strings.TrimSpace(hSrc)
				citation = fmt.Sprintf(fstr[f.kind], hSrc)
			} else {
				citation = fmt.Sprintf(fstr[f.kind], src)
			}
			t.Body = replace(t.Body, citation, f.i+offset, 1)
			offset += len(citation) - 1
		case anchorClose:
			if t.Body[f.w+offset] == ')' {
				t.Body = replace(t.Body, fstr[f.kind], f.extra+offset, f.w-f.extra+1)
				offset += len(fstr[f.kind]) - (f.w - f.extra + 1)
			} else {
				t.Body = replace(t.Body, fstr[f.kind], f.w+offset, 1)
				offset += len(fstr[f.kind]) - 1
			}
		default:
			t.Body = replace(t.Body, fstr[f.kind], f.i+offset-f.w+1, f.w)
			offset += len(fstr[f.kind]) - f.w
		}
	}
	return w.Write([]byte(t.Body))
}

func (g *Generator) list(l *ast.List, w io.Writer) error {
	currTab := 0
	w.Write([]byte("<ul>"))
	for _, li := range l.Items {
		diffTab := li.NTab - currTab
		currTab = li.NTab
		for diffTab > 0 {
			w.Write([]byte("<ul>"))
			diffTab--
		}
		for diffTab < 0 {
			w.Write([]byte("</ul>"))
			diffTab++
		}
		if len(li.Label) != 0 {
			w.Write([]byte("<li>"))
			fmt.Fprintf(w, "<span>%s</span>", li.Label)
		} else {
			w.Write([]byte("<li class=\"bullet\">"))
		}
		g.text(&li.Text, w)
		w.Write([]byte("</li>"))
	}
	for currTab >= 0 {
		w.Write([]byte("</ul>"))
		currTab--
	}
	return nil
}

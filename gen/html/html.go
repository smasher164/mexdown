package html

import (
	"context"
	"fmt"
	"html"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/smasher164/mexdown/ast"
	"github.com/smasher164/mexdown/gen"
)

type Genner struct {
	File     *ast.File
	Ctx      context.Context
	Stderr   io.Writer
	out      *syncBuffer
	launched bool
	done     uint32
}

func (g *Genner) Read(p []byte) (n int, err error) {
	if !g.launched {
		g.launched = true
		g.out = new(syncBuffer)
		go func() {
			if _, err := g.WriteTo(g.out); err != nil {
				g.out.SetError(err)
			}
			atomic.StoreUint32(&g.done, 1)
		}()
	}
	// prevent waiting read/writes from eof'ing
	n, err = g.out.Read(p)
	if atomic.LoadUint32(&g.done) == 0 && err == io.EOF {
		err = nil
	}
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

func (g *Genner) WriteTo(w io.Writer) (n int64, err error) {
	cw := &stickyCountWriter{0, nil, w}
	ctx := g.Ctx
	if ctx == nil {
		ctx = context.TODO()
	}
	for i := range g.File.List {
		select {
		case <-ctx.Done():
			return cw.n, cw.err
		default:
			switch t := g.File.List[i].(type) {
			case *ast.Paragraph:
				if t.Body != "" {
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
				if t.Command == "" {
					fmt.Fprintf(cw, "<pre>%s</pre>", html.EscapeString(t.Raw))
				} else {
					c := &gen.Command{Ctx: g.Ctx, Stderr: g.Stderr}
					if err := c.Gen(t, cw); err != nil {
						return cw.n, err
					}
				}
			}
		}
	}
	return cw.n, cw.err
}

func replace(s, r string, pos, width int) string {
	return s[:pos] + r + s[pos+width:]
}

func open(tag int) bool {
	return tag&1 == 0
}

const (
	cite = iota
	citeClose
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
	raw
	rawClose
)

var fstr = [...]string{
	cite:               "<a href=%q>",
	citeClose:          "</a>",
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
	raw:                "<code>",
	rawClose:           "</code>",
}

func (h *Genner) text(t *ast.Text, w io.Writer) (n int, err error) {
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
		// escape raw text
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
			rep = append(rep, repl{f.Beg, f.End, cite, 0})
			begClose := f.Beg
			if t.Body[f.End] == ')' {
				begClose += strings.Index(t.Body[f.Beg:], "](")
			}
			rep = append(rep, repl{f.End, f.End, citeClose, begClose})
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
			rep = append(rep, repl{f.Beg, 1, raw, 0})
			rep = append(rep, repl{f.End, 1, rawClose, 0})
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
				if rep[lower].kind != cite && open(rep[lower].kind) && (rep[current].kind-rep[lower].kind) != 1 {
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
		case cite:
			del := strings.Index(t.Body[f.i+offset:], "](") + f.i + offset
			src := t.Body[del+2 : f.w+offset]
			var citation string
			if t.Body[f.w+offset] != ')' {
				citation = fmt.Sprintf(fstr[f.kind], t.Body[f.i+offset+1:f.w+offset])
			} else if hSrc, ok := h.File.Cite[src]; ok {
				hSrc = strings.TrimSpace(hSrc)
				citation = fmt.Sprintf(fstr[f.kind], hSrc)
			} else {
				citation = fmt.Sprintf(fstr[f.kind], src)
			}
			t.Body = replace(t.Body, citation, f.i+offset, 1)
			offset += len(citation) - 1
		case citeClose:
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

func (h *Genner) list(l *ast.List, w io.Writer) error {
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
		if li.Label != "" {
			w.Write([]byte("<li>"))
			fmt.Fprintf(w, "<span>%s</span>", li.Label)
		} else {
			w.Write([]byte("<li class=\"bullet\">"))
		}
		h.text(&li.Text, w)
		w.Write([]byte("</li>"))
	}
	for currTab >= 0 {
		w.Write([]byte("</ul>"))
		currTab--
	}
	return nil
}

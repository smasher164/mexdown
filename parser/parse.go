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

// Package parser implements a parser for mexdown source. It takes in an io.Reader
// as input and outputs an *ast.File.
//
// It is the responsibility of the generator to parse
// commands attached to directive nodes.
//
// The parser adheres to the following grammar for mexdown source files:
//
//      unicode_char = /* an arbitrary Unicode code point except newline */ .
//      newline      = /* the Unicode code point U+000A */ .
//      tab          = /* the Unicode code point U+0009 */ .
//      octothorpe   = /* the Unicode code point U+0023 */ .
//      backtick     = /* the Unicode code point U+0060 */ .
//      hyphen       = /* the Unicode code point U+002D */ .
//      asterisk     = /* the Unicode code point U+002A */ .
//      lbrack       = /* the Unicode code point U+005B */ .
//      rbrack       = /* the Unicode code point U+005D */ .
//      lparen       = /* the Unicode code point U+0028 */ .
//      rparen       = /* the Unicode code point U+0029 */ .
//      underscore   = /* the Unicode code point U+005F */ .
//      colon        = /* the Unicode code point U+003A */ .
//
//      citation = lbrack text rbrack colon string .
//      paragraph = text .
//      list_item = { tab } hyphen [ lbrack text rbrack ] text .
//      list = { list_item newline } [ list_item ] .
//      string = { unicode_char | newline } .
//      command = unicode_char { unicode_char } .
//      dirbody = backtick dirbody backtick | [ command ] newline string .
//      directive = backtick backtick backtick dirbody backtick backtick backtick .
//      text = unicode_char { unicode_char } |
//             lbrack text rbrack lparen text rparen |
//             asterisk text asterisk |
//             asterisk asterisk text asterisk asterisk |
//             underscore text underscore |
//             hyphen hyphen text hyphen hyphen |
//             backtick text backtick .
//      header = octothorpe { octothorpe } text .
//      statement = header | directive | list | paragraph | citation .
//      source_file = { statement [ newline ] [ newline ] } .
//
// In the relevant context, the following characters are escaped (in Go syntax):
//
//      '\\', '#', '`', '-', '*', '[', ']', '(', ')', '_'
//
package parser // import "akhil.cc/mexdown/parser"

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"akhil.cc/mexdown/ast"
)

// MustParse is like Parse but panics if the source cannot be parsed.
func MustParse(src io.Reader) (file *ast.File) {
	f, err := Parse(src)
	if err != nil {
		panic("Parse error: " + err.Error())
	}
	return f
}

// Parse parses the source and if successful, returns its corresponding AST structure.
// A generator can be used to transform the returned AST into another format.
func Parse(src io.Reader) (f *ast.File, err error) {
	p := &parser{
		errors: []error{},
		b:      bufio.NewReader(src),
		cite:   make(map[string]string),
	}
	// source_file = { statement [ newline ] [ newline ] } .
	f = &ast.File{List: []ast.Stmt{}}
	p.next()
	for p.r != eof || p.st != nil {
		f.List = append(f.List, p.stmt())
	}
	f.Cite = p.cite
	// combine consecutive paragraphs
	for i, j := 0, 1; i < len(f.List) && j < len(f.List); i, j = i+1, j+1 {
		pi, _ := f.List[i].(*ast.Paragraph)
		pj, _ := f.List[j].(*ast.Paragraph)
		// validate that they are not empty
		sub := 0
		if pi != nil && strings.TrimSpace(pi.Body) == "" {
			// delete
			copy(f.List[i:], f.List[i+1:])
			f.List[len(f.List)-1] = nil
			f.List = f.List[:len(f.List)-1]
			sub++
		}
		if pj != nil && strings.TrimSpace(pj.Body) == "" {
			// delete
			copy(f.List[j-sub:], f.List[j-sub+1:])
			f.List[len(f.List)-1] = nil
			f.List = f.List[:len(f.List)-1]
			sub++
		}
		if sub > 0 {
			i, j = i-sub, j-sub
		} else if pi != nil && pj != nil {
			// body
			par := &ast.Paragraph{Body: pi.Body + pj.Body, Format: pi.Format}
			f.List[i] = par
			// remove jth
			copy(f.List[j:], f.List[j+1:])
			f.List[len(f.List)-1] = nil
			f.List = f.List[:len(f.List)-1]
			// subtract indices
			i, j = i-1, j-1
		}
	}
	// Parse formats for paragraph
	for i := range f.List {
		pi, _ := f.List[i].(*ast.Paragraph)
		if pi != nil {
			rdr := io.MultiReader(strings.NewReader(pi.Body+string(eof)+string(p.r)), p.b)
			p.b = bufio.NewReader(rdr)
			p.next()
			txt := p.text(eof)
			pi.Body = txt.Body
			pi.Format = txt.Format
		}
	}
	var es strings.Builder
	for _, e := range p.errors {
		es.WriteString(e.Error())
		es.WriteString("\n")
	}
	if len(p.errors) > 0 {
		err = errors.New(es.String())
	}
	return f, err
}

const eof = -1

type parser struct {
	errors []error
	b      *bufio.Reader
	r      rune
	st     ast.Stmt
	cite   map[string]string
}

// statement = header | directive | list | paragraph | citation .
func (p *parser) stmt() ast.Stmt {
	if p.st != nil {
		temp := p.st
		p.st = nil
		return temp
	}
	switch p.r {
	case '#':
		return p.header()
	case '`':
		return p.directive()
	case '[':
		return p.citation()
	case '-':
		l, st := p.list()
		if l.Items == nil {
			return st
		}
		p.st = st
		return l
	default:
		return p.paragraph("")
	}
}

// header = octothorpe { octothorpe } text .
func (p *parser) header() *ast.Header {
	var hdr ast.Header
	hdr.NThorpe = 1
	for p.next() == '#' {
		hdr.NThorpe++
	}
	hdr.Text = p.text('\n')
	return &hdr
}

// text = unicode_char { unicode_char } |
//        lbrack text rbrack lparen text rparen |
//        asterisk text asterisk |
//        asterisk asterisk text asterisk asterisk |
//        underscore text underscore |
//        hyphen hyphen text hyphen hyphen |
//        backtick text backtick .
func (p *parser) text(end rune) ast.Text {
	type token struct {
		s   string
		pos int
	}
	var (
		pos    int
		tokens []token
		format []ast.Format
		buf    strings.Builder
		inRaw  bool
	)
	// maintain a stack of tokens that correspond to formatting tags inside text
	for p.r != end && p.r != eof {
		switch p.r {
		case '*':
			buf.WriteRune(p.r)
			s := "*"
			if len(tokens) > 0 {
				top := tokens[len(tokens)-1]
				if top.s == "*" && (pos-top.pos) == 1 {
					s = "**"
					tokens = tokens[:len(tokens)-1]
				}
				if top.s == "**" && (pos-top.pos) == 1 {
					s = "***"
					tokens = tokens[:len(tokens)-1]
				}
			}
			tokens = append(tokens, token{s, pos})
		case '_':
			buf.WriteRune(p.r)
			tokens = append(tokens, token{"_", pos})
		case '[':
			buf.WriteRune(p.r)
			tokens = append(tokens, token{"[", pos})
		case ']':
			buf.WriteRune(p.r)
			tokens = append(tokens, token{"]", pos})
		case '(':
			buf.WriteRune(p.r)
			if len(tokens) > 0 {
				top := tokens[len(tokens)-1]
				if top.s == "]" && (pos-top.pos) == 1 {
					tokens[len(tokens)-1] = token{"](", pos}
				}
			}
		case ')':
			buf.WriteRune(p.r)
			tokens = append(tokens, token{")", pos})
		case '`':
			buf.WriteRune(p.r)
			tokens = append(tokens, token{"`", pos})
			inRaw = !inRaw
		case '-':
			buf.WriteRune(p.r)
			s := "-"
			if len(tokens) > 0 {
				top := tokens[len(tokens)-1]
				if top.s == "-" && (pos-top.pos) == 1 {
					s = "--"
					tokens = tokens[:len(tokens)-1]
				}
			}
			tokens = append(tokens, token{s, pos})
		case '\\':
			p.next()
			if !escapable(p.r) || inRaw {
				pos++
				buf.WriteRune('\\')
			}
			buf.WriteRune(p.r)
		default:
			buf.WriteRune(p.r)
			if len(tokens) > 0 && tokens[len(tokens)-1].s == "-" {
				tokens = tokens[:len(tokens)-1]
			}
		}
		p.next()
		pos++
	}
	p.next()

	// highest precedence is raw,
	// so you can do a pass to eliminate everything that is between raw tags
	//  if there is an odd number of backtick characters, then the last backtick
	//    character is not a delimeter for a raw tag
	//  you can append these format asts then
	ib := -1
	for i := 0; i < len(tokens); i++ {
		if tokens[i].s == "`" {
			if ib == -1 {
				ib = i
			}
			if ib < i {
				format = append(format, ast.Format{
					Kind: ast.Raw,
					Beg:  tokens[ib].pos,
					End:  tokens[i].pos,
				})
				disp := len(tokens)
				tokens = append(tokens[:ib], tokens[i+1:]...)
				disp -= len(tokens)
				i -= disp
				ib = -1
			}
		}
	}

	// assumes slice doesn't have high-prec operators like citations or raw quotes.
	lowprec := func(tokens []token) {
		idx := []int{-1, -1, -1, -1, -1}
		for i := range tokens {
			switch tokens[i].s {
			case "*":
				if idx[0] == -1 {
					idx[0] = i
				}
				if idx[0] < i {
					format = append(format, ast.Format{
						Kind: ast.Italic,
						Beg:  tokens[idx[0]].pos,
						End:  tokens[i].pos,
					})
					idx[0] = -1
					idx[1] = -1
					idx[2] = -1
				}
			case "**":
				if idx[1] == -1 {
					idx[1] = i
				}
				if idx[1] < i {
					format = append(format, ast.Format{
						Kind: ast.Bold,
						Beg:  tokens[idx[1]].pos,
						End:  tokens[i].pos,
					})
					idx[0] = -1
					idx[1] = -1
					idx[2] = -1
				}
			case "***":
				if idx[2] == -1 {
					idx[2] = i
				}
				if idx[2] < i {
					format = append(format, ast.Format{
						Kind: ast.BoldItalic,
						Beg:  tokens[idx[2]].pos,
						End:  tokens[i].pos,
					})
					idx[0] = -1
					idx[1] = -1
					idx[2] = -1
				}
			case "--":
				if idx[3] == -1 {
					idx[3] = i
				}
				if idx[3] < i {
					format = append(format, ast.Format{
						Kind: ast.Strikethrough,
						Beg:  tokens[idx[3]].pos,
						End:  tokens[i].pos,
					})
					idx[3] = -1
				}
			case "_":
				if idx[4] == -1 {
					idx[4] = i
				}
				if idx[4] < i {
					format = append(format, ast.Format{
						Kind: ast.Underline,
						Beg:  tokens[idx[4]].pos,
						End:  tokens[i].pos,
					})
					idx[4] = -1
				}
			}
		}
	}

	// now you know that you don’t have to check for raw. only problem now is citation
	// on a second pass, for each citation, parse its inner section. remove all of those tokens.
	cit := []int{-1, -1, -1, -1}
	for i := 0; i < len(tokens); i++ {
		switch tokens[i].s {
		case "[":
			if cit[0] == -1 {
				cit[0] = i
			}
			if cit[0] < i {
				cit[0] = i
				cit[1] = -1
				cit[2] = -1
				cit[3] = -1
			}
		case "]":
			if cit[1] == -1 {
				cit[1] = i
			}
			if cit[0] != -1 {
				format = append(format, ast.Format{
					Kind: ast.Cite,
					Beg:  tokens[cit[0]].pos,
					End:  tokens[i].pos,
				})
				// we can now parse the inside of the brackets
				subtok := tokens[cit[0] : i+1]
				lowprec(subtok)
				disp := len(tokens)
				tokens = append(tokens[:cit[0]], tokens[i+1:]...)
				disp -= len(tokens)
				i -= disp
				cit[0] = -1
				cit[1] = -1
			}
		case "](":
			if cit[2] == -1 {
				cit[2] = i
			}
			if cit[0] == -1 {
				cit[2] = -1
			}
		case ")":
			if cit[3] == -1 {
				cit[3] = i
			}
			if cit[0] != -1 && cit[2] != -1 {
				format = append(format, ast.Format{
					Kind: ast.Cite,
					Beg:  tokens[cit[0]].pos,
					End:  tokens[i].pos,
				})
				subtok := tokens[cit[0] : cit[2]+1]
				lowprec(subtok)
				disp := len(tokens)
				tokens = append(tokens[:cit[0]], tokens[i+1:]...)
				disp -= len(tokens)
				i -= disp
				cit[0] = -1
				cit[1] = -1
				cit[2] = -1
			}
		}
	}
	if cit[0] > -1 && cit[2] > cit[0] {
		format = append(format, ast.Format{
			Kind: ast.Cite,
			Beg:  tokens[cit[0]].pos,
			End:  tokens[cit[2]].pos - 1,
		})
	}
	// in the last pass emit everything else
	lowprec(tokens)
	return ast.Text{
		Format: format,
		Body:   buf.String(),
	}
}

// dirbody = backtick dirbody backtick | [ command ] newline string .
// directive = backtick backtick backtick dirbody backtick backtick backtick .
func (p *parser) directive() ast.Stmt {
	if p.next() != '`' {
		return p.paragraph("`")
	}
	if p.next() != '`' {
		return p.paragraph("``")
	}
	prefix := "```"
	for p.next() == '`' {
		prefix += "`"
	}
	cmd := strings.TrimSuffix(p.line(nil), "\n")
	if cmd != "" {
		cmd += "\n"
	}
	var buf strings.Builder
	for {
		l := strings.TrimSuffix(p.line(nil), "\n")
		if strings.HasPrefix(l, prefix) {
			if len(strings.TrimSpace(l[len(prefix):])) != 0 {
				p.errorf("Cannot have text on the same line that a directive is terminated: %s\n", l)
			}
			break
		}
		if p.r == eof {
			out := buf.String()
			if len(out) > 10 {
				out = out[10:]
			}
			p.errorf("Directive is not terminated: %s", out)
			break
		}
		buf.WriteString(l + "\n")
	}
	return &ast.Directive{
		Command: cmd,
		Raw:     buf.String(),
	}
}

// list = { list_item newline } [ list_item ] .
func (p *parser) list() (*ast.List, ast.Stmt) {
	var (
		l  ast.List
		li ast.ListItem
		e  error
	)
	for e != notList {
		li, e = p.listItem()
		// fmt.Println("called", li)
		if e == notList {
			break
		}
		l.Items = append(l.Items, li)
	}
	if len(li.Text.Body) > 0 {
		if li.Text.Body[0] == '-' {
			return &l, p.paragraph(li.Text.Body)
		}
		rdr := io.MultiReader(strings.NewReader(li.Text.Body+string(p.r)), p.b)
		p.b = bufio.NewReader(rdr)
		p.next()
	}
	return &l, p.stmt()
}

var notList = errors.New("not list item")

// returning nil means paragraph
// list_item = { tab } hyphen [ lbrack text rbrack ] text .
func (p *parser) listItem() (ast.ListItem, error) {
	var li ast.ListItem
	for p.r == '\t' {
		li.NTab++
		p.next()
	}
	// fmt.Println("p.r", string(p.r))
	if p.r != '-' {
		for i := 0; i < li.NTab; i++ {
			li.Text.Body += "\t"
		}
		return li, notList
	}
	p.next()
	if p.r == '-' {
		for i := 0; i < li.NTab; i++ {
			li.Text.Body += "\t"
		}
		li.Text.Body += "-"
		return li, notList
	}
	if p.r == '[' {
		p.next()
		li.Label = p.str(func(r rune) bool { return r == ']' }, func(r rune) bool { return r == '\\' || r == ']' }, nil)
		if p.r != ']' {
			p.errorf("List item's label does not have a closing bracket: %s", "["+li.Label)
		}
		p.next()
	}
	ln := strings.TrimSuffix(p.line(nil), "\n") // w/o '\n' at the end
	// Combine consecutive list items
	for {
		l := strings.TrimSuffix(p.line(nil), "\n") // w/o '\n' at the end
		tr := strings.TrimSpace(l)
		if len(tr) == 0 || tr[0] == '-' {
			// new list or new item
			ln += string(eof) + l + "\n"
			break
		}
		// same item
		ln += " " + l
	}
	rdr := io.MultiReader(strings.NewReader(ln+string(p.r)), p.b)
	p.b = bufio.NewReader(rdr)
	p.next()
	li.Text = p.text(eof)
	return li, nil
}

func (p *parser) paragraph(before string) *ast.Paragraph {
	if len(before) > 0 {
		rdr := io.MultiReader(strings.NewReader(before+string(p.r)), p.b)
		p.b = bufio.NewReader(rdr)
		p.next()
	}
	b := p.line(nil)
	b += p.str(func(r rune) bool { return r != '\n' }, func(r rune) bool { return r == '\\' }, nil)
	par := ast.Paragraph{
		Format: nil,
		Body:   b,
	}
	return &par
}

func (p *parser) next() rune {
	r, size, err := p.b.ReadRune()
	if err == io.EOF || size == 0 || r == utf8.RuneError {
		r = eof
	}
	p.r = r
	return r
}

// str reads all input up to, but not including end.
// Does not advance pointer in input past end.
// Calls f to determine whether or not to write.
func (p *parser) str(end func(rune) bool, esc, f func(rune) bool) string {
	var buf strings.Builder
	var eb strings.Builder
	for {
		escaped := false
		if p.r == '\\' && esc != nil {
			eb.WriteRune('\\')
			p.next()
			if !esc(p.r) {
				buf.WriteRune('\\')
			} else {
				escaped = true
			}
		}
		if (end(p.r) && !escaped) || p.r == eof {
			break
		}
		if f == nil || f(p.r) {
			buf.WriteRune(p.r)
			eb.WriteRune(p.r)
		}
		p.next()
	}
	if p.r == eof {
		return eb.String()
	}
	return buf.String()
}

func escapable(r rune) bool {
	switch r {
	case '\\', '#', '`', '-', '*', '[', ']', '(', ')', '_':
		return true
	}
	return false
}

// citation = lbrack text rbrack colon string .
func (p *parser) citation() ast.Stmt {
	p.next()
	label := p.str(func(r rune) bool { return r == ']' }, func(r rune) bool { return r == '\\' || r == ']' }, nil)
	if p.r != ']' {
		return p.paragraph("[" + label)
	}
	if p.next() != ':' {
		return p.paragraph("[" + label + "]")
	}
	p.next()
	src := strings.TrimSuffix(p.line(nil), "\n")
	p.cite[label] = src
	return &ast.Citation{
		Label: label,
		Src:   src,
	}
}

func (p *parser) line(esc func(r rune) bool) string {
	l := p.str(func(r rune) bool { return r == '\n' }, esc, func(r rune) bool { return p.r != '\r' })
	if p.r == '\n' {
		l += "\n"
	}
	p.next()
	return l
}

func (p *parser) errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Errorf(format, args...))
}

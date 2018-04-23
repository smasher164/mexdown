package parse

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/smasher164/mexdown/ast"
)

const eof = -1

type parser struct {
	errors []error
	b      *bufio.Reader
	r      rune
	st     ast.Stmt
	cite   map[string]string
}

func MustParse(r io.Reader) *ast.File {
	f := Parse(r)
	if len(f.Errors) > 0 {
		panic(concatErr(f.Errors))
	}
	return f
}

func Parse(r io.Reader) *ast.File {
	p := &parser{
		errors: []error{},
		b:      bufio.NewReader(r),
		cite:   make(map[string]string),
	}
	f := p.parse()
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
			par := &ast.Paragraph{Body: pi.Body + "\n" + pj.Body, Format: pi.Format}
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
	return f
}

func concatErr(errs []error) error {
	var buf strings.Builder
	for _, e := range errs {
		buf.WriteString(e.Error() + "\n")
	}
	return errors.New(buf.String())
}

func (p *parser) parse() *ast.File {
	f := &ast.File{List: []ast.Stmt{}}
	p.next()
	for p.r != eof || p.st != nil {
		f.List = append(f.List, p.stmt())
	}
	f.Errors = p.errors
	f.Cite = p.cite
	return f
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
func (p *parser) str(end rune, esc, f func(rune) bool) string {
	var buf strings.Builder
	for {
		if p.r == '\\' && esc != nil {
			p.next()
			if !esc(p.r) {
				buf.WriteRune('\\')
			}
		}
		if p.r == end || p.r == eof {
			break
		}
		if f == nil || f(p.r) {
			buf.WriteRune(p.r)
		}
		p.next()
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
	// 	if there is an odd number of backtick characters, then the last backtick character is not a delimeter for a raw tag
	// 	you can append these format asts then
	ib := -1
	for i := 0; i < len(tokens); i++ {
		if tokens[i].s == "`" {
			if ib == -1 {
				ib = i
			}
			if ib < i {
				format = append(format, ast.Format{ast.Raw, tokens[ib].pos, tokens[i].pos})
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
					format = append(format, ast.Format{ast.Italic, tokens[idx[0]].pos, tokens[i].pos})
					idx[0] = -1
					idx[1] = -1
					idx[2] = -1
				}
			case "**":
				if idx[1] == -1 {
					idx[1] = i
				}
				if idx[1] < i {
					format = append(format, ast.Format{ast.Bold, tokens[idx[1]].pos, tokens[i].pos})
					idx[0] = -1
					idx[1] = -1
					idx[2] = -1
				}
			case "***":
				if idx[2] == -1 {
					idx[2] = i
				}
				if idx[2] < i {
					format = append(format, ast.Format{ast.BoldItalic, tokens[idx[2]].pos, tokens[i].pos})
					idx[0] = -1
					idx[1] = -1
					idx[2] = -1
				}
			case "--":
				if idx[3] == -1 {
					idx[3] = i
				}
				if idx[3] < i {
					format = append(format, ast.Format{ast.Strikethrough, tokens[idx[3]].pos, tokens[i].pos})
					idx[3] = -1
				}
			case "_":
				if idx[4] == -1 {
					idx[4] = i
				}
				if idx[4] < i {
					format = append(format, ast.Format{ast.Underline, tokens[idx[4]].pos, tokens[i].pos})
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
				format = append(format, ast.Format{ast.Cite, tokens[cit[0]].pos, tokens[i].pos})
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
				format = append(format, ast.Format{ast.Cite, tokens[cit[0]].pos, tokens[i].pos})
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
	// in the last pass emit everything else
	lowprec(tokens)
	return ast.Text{
		Format: format,
		Body:   buf.String(),
	}
}

func (p *parser) paragraph(before string) *ast.Paragraph {
	if len(before) > 0 {
		rdr := io.MultiReader(strings.NewReader(before+string(p.r)), p.b)
		p.b = bufio.NewReader(rdr)
		p.next()
	}
	par := ast.Paragraph{
		Format: nil,
		Body:   p.line(nil),
	}
	return &par
}

func (p *parser) line(esc func(r rune) bool) string {
	l := p.str('\n', esc, func(r rune) bool { return p.r != '\r' })
	p.next()
	return l
}

func (p *parser) errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Errorf(format, args...))
}

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

func (p *parser) header() *ast.Header {
	var hdr ast.Header
	hdr.NThorpe = 1
	for p.next() == '#' {
		hdr.NThorpe++
	}
	hdr.Text = p.text('\n')
	return &hdr
}

// directive = backtick backtick backtick { backtick } [ command ] newline
// 			   string
// 			   backtick backtick backtick { backtick } .
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
	cmd := p.line(nil)
	if cmd != "" {
		cmd += "\n"
	}
	var buf strings.Builder
	for {
		l := p.line(nil)
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

// citation = lbrack text rbrack colon string .
func (p *parser) citation() ast.Stmt {
	p.next()
	label := p.str(']', func(r rune) bool { return r == '\\' || r == ']' }, nil)
	if p.r != ']' {
		p.errorf("Citation does not have a closing bracket: %s", "["+label)
	}
	if p.next() != ':' {
		return p.paragraph("[" + label + "]")
	}
	p.next()
	src := p.line(nil)
	p.cite[label] = src
	return &ast.Citation{
		Label: label,
		Src:   src,
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
		li.Label = p.str(']', func(r rune) bool { return r == '\\' || r == ']' }, nil)
		if p.r != ']' {
			p.errorf("List item's label does not have a closing bracket: %s", "["+li.Label)
		}
		p.next()
	}
	ln := p.line(nil) + "\n"
	// Combine consecutive list items
	for {
		l := p.line(nil) + "\n"
		tr := strings.TrimSpace(l)
		if len(tr) == 0 {
			ln += string(eof)
			break
		}
		if tr[0] == '-' {
			ln += string(eof) + l
			break
		}
		ln += l
	}
	rdr := io.MultiReader(strings.NewReader(ln+string(p.r)), p.b)
	p.b = bufio.NewReader(rdr)
	p.next()
	li.Text = p.text(eof)
	return li, nil
}

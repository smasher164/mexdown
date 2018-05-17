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

// Package ast declares the structures used to represent mexdown syntax trees.
package ast // import "akhil.cc/mexdown/ast"

// All Node types implement the Node interface.
//go:generate sumgen Node = *File | *Header | *Directive | *List | ListItem | *Paragraph | Text
type Node interface {
	node()
}

// All statement nodes implement the Stmt interface.
//go:generate sumgen Stmt = *Header | *Directive | *List | *Paragraph | *Citation
type Stmt interface {
	Node
	stmt()
}

// File represents a mexdown source file. It stores a list of statements representing
// the source text and citations referenced by any links in the source.
type File struct {
	List []Stmt
	Cite map[string]string
}

// A Header statement represents a multi-level section heading.
type Header struct {
	NThorpe int // Number of preceding octothorpes '#'
	Text    Text
}

// A Directive statement represents either raw (preformatted) text, or an input string to pass into a command.
type Directive struct {
	Command string
	Raw     string
}

// A List statement represents a sequence of list items.
type List struct {
	Items []ListItem
}

// A ListItem node represents text preceded by a label.
type ListItem struct {
	NTab  int // Number of preceding tab characters '\t'
	Label string
	Text  Text
}

// A Citation statement represents the corresponding source for a cited label.
// This label can be referenced in links.
type Citation struct {
	Label string
	Src   string
}

// A Paragraph statement represents a body of text with formatting applied.
type Paragraph struct {
	Format []Format
	Body   string
}

// A Text node represents an arbitrary body of text with formatting applied.
type Text Paragraph

// Format holds position and type information for a body of text.
type Format struct {
	Kind FType
	Beg  int
	End  int
}

// An FType is the set of valid formats applied to text.
type FType int

const (
	Cite          FType = iota // [src] or [label](src)
	Italic                     // *text*
	Bold                       // **text**
	BoldItalic                 // ***text***
	Underline                  // _text_
	Strikethrough              // --text--
	Raw                        // `text`
)

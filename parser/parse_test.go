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

// Tests for parse.go
package parser_test

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"akhil.cc/mexdown/parser"

	"akhil.cc/mexdown/ast"
	"github.com/sanity-io/litter"
)

type smallcase struct {
	in   string
	want ast.File
	werr error
}

func fileEquals(want, got ast.File) bool {
	if len(want.List) != len(got.List) {
		return false
	}
	if !reflect.DeepEqual(want.Cite, got.Cite) {
		if len(want.Cite) != 0 && len(got.Cite) != 0 {
			return false
		}
	}
	for i := range want.List {
		v1 := reflect.ValueOf(want.List[i])
		v2 := reflect.ValueOf(got.List[i])
		// Check type equality
		if v1.Type() != v2.Type() {
			return false
		}
		// Dereference both pointers
		v1, v2 = reflect.Indirect(v1), reflect.Indirect(v2)
		// Get their interface values
		i1, i2 := v1.Interface(), v2.Interface()
		// Perform a deep comparison
		if p2, ok1 := i2.(ast.Paragraph); ok1 {
			sort.Slice(p2.Format, func(i, j int) bool {
				return p2.Format[i].Beg < p2.Format[j].Beg
			})
			i2 = p2
		} else if t2, ok2 := i2.(ast.Text); ok2 {
			sort.Slice(t2.Format, func(i, j int) bool {
				return t2.Format[i].Beg < t2.Format[j].Beg
			})
			i2 = t2
		}
		if !reflect.DeepEqual(i1, i2) {
			return false
		}
	}
	return true
}

var overlapSmall = []smallcase{
	{"abc***def_ghi***jkl_", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.BoldItalic, Beg: 5, End: 15},
					ast.Format{Kind: ast.Underline, Beg: 9, End: 19},
				},
				Body: "abc***def_ghi***jkl_",
			},
		}}, nil,
	},
	{"*aa**bbb***c*dddd**ee***", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Italic, Beg: 0, End: 12},
				},
				Body: "*aa**bbb***c*dddd**ee***",
			},
		}}, nil,
	},
	{"**a_bbb--cc**dddd_e--", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Bold, Beg: 1, End: 12},
					ast.Format{Kind: ast.Underline, Beg: 3, End: 17},
					ast.Format{Kind: ast.Strikethrough, Beg: 8, End: 20},
				},
				Body: "**a_bbb--cc**dddd_e--",
			},
		}}, nil,
	},
	{"*a`b*c`d*", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Italic, Beg: 0, End: 8},
					ast.Format{Kind: ast.Raw, Beg: 2, End: 6},
				},
				Body: "*a`b*c`d*",
			},
		}}, nil,
	},
	{"*a[*a`b*c`d*]b*", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Italic, Beg: 0, End: 14},
					ast.Format{Kind: ast.Cite, Beg: 2, End: 12},
					ast.Format{Kind: ast.Italic, Beg: 3, End: 11},
					ast.Format{Kind: ast.Raw, Beg: 5, End: 9},
				},
				Body: "*a[*a`b*c`d*]b*",
			},
		}}, nil,
	},
	{"*a[*a`b*c`d*](url)b*", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Italic, Beg: 0, End: 19},
					ast.Format{Kind: ast.Cite, Beg: 2, End: 17},
					ast.Format{Kind: ast.Italic, Beg: 3, End: 11},
					ast.Format{Kind: ast.Raw, Beg: 5, End: 9},
				},
				Body: "*a[*a`b*c`d*](url)b*",
			},
		}}, nil,
	},
	{"`[`]`", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Raw, Beg: 0, End: 2},
				},
				Body: "`[`]`",
			},
		}}, nil,
	},
	{"_hi[_hello_]bye_", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Underline, Beg: 0, End: 15},
					ast.Format{Kind: ast.Cite, Beg: 3, End: 11},
					ast.Format{Kind: ast.Underline, Beg: 4, End: 10},
				},
				Body: "_hi[_hello_]bye_",
			},
		}}, nil,
	},
}

func TestOverlap(t *testing.T) {
	litCfg := litter.Options{
		Compact:           true,
		StripPackageNames: false,
		HidePrivateFields: false,
		Separator:         " ",
	}
	for i, test := range overlapSmall {
		got, err := parser.Parse(strings.NewReader(test.in))
		if wes, es := fmt.Sprint(test.werr), fmt.Sprint(err); es != wes || !fileEquals(test.want, *got) {
			t.Errorf("case %d, in %q,\nwant %s,\ngot %s,\nwant err %s,\ngot err %s", i, test.in, litCfg.Sdump(test.want), litCfg.Sdump(*got), wes, es)
		}
	}
}

var escapeSmall = []smallcase{
	// '\\', '#', '`', '-', '*', '[', ']', '(', ')', '_':
	/*
		\c		No \tab
		\`		\`Not Raw`
		\\		\\`Raw`
		\#		\#Not Header
		\-		\--Not Strikethrough--
		\*		\***No Format***
		\[		\[Not Cite]
		\]		[Not Cite\]
		\(		[Direct Cite]\(Not Sourced)
		\)		[Direct Cite](Not Sourced\)
		\_		\_Not underlined_
	*/
	{`No \tab`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: nil,
				Body:   `No \tab`,
			},
		}}, nil,
	},
	{"\\`Not Raw`", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: nil,
				Body:   "`Not Raw`",
			},
		}}, nil,
	},
	{"\\\\`Raw`", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Raw, Beg: 1, End: 5},
				},
				Body: "\\`Raw`",
			},
		}}, nil,
	},
	{`\#Not Header`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: nil,
				Body:   `#Not Header`,
			},
		}}, nil,
	},
	{`\--Not Strikethrough--`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: nil,
				Body:   `--Not Strikethrough--`,
			},
		}}, nil,
	},
	{`\***No Format***`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: nil,
				Body:   `***No Format***`,
			},
		}}, nil,
	},
	{`\[Not Cite]`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: nil,
				Body:   `[Not Cite]`,
			},
		}}, nil,
	},
	{`[Not Cite\]`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: nil,
				Body:   `[Not Cite]`,
			},
		}}, nil,
	},
	{`[Direct Cite]\(Not Sourced)`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Cite, Beg: 0, End: 12},
				},
				Body: `[Direct Cite](Not Sourced)`,
			},
		}}, nil,
	},
	{`[Direct Cite](Not Sourced\)`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Cite, Beg: 0, End: 12},
				},
				Body: `[Direct Cite](Not Sourced)`,
			},
		}}, nil,
	},
	{`\_Not underlined_`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: nil,
				Body:   `_Not underlined_`,
			},
		}}, nil,
	},
}

func TestEscape(t *testing.T) {
	litCfg := litter.Options{
		Compact:           true,
		StripPackageNames: false,
		HidePrivateFields: false,
		Separator:         " ",
	}
	for i, test := range escapeSmall {
		got, err := parser.Parse(strings.NewReader(test.in))
		if wes, es := fmt.Sprint(test.werr), fmt.Sprint(err); es != wes || !fileEquals(test.want, *got) {
			t.Errorf("case %d, in %q,\nwant %s,\ngot %s,\nwant err %s,\ngot err %s", i, test.in, litCfg.Sdump(test.want), litCfg.Sdump(*got), wes, es)
		}
	}
}
func TestCombineListItem(t *testing.T) {

}
func TestCombineParagraph(t *testing.T) {

}
func TestUnicode(t *testing.T) {

}

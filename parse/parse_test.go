package parse

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/sanity-io/litter"
	"github.com/smasher164/mexdown/ast"
)

type smallcase struct {
	in   string
	want ast.File
}

func fileEquals(want, got ast.File) bool {
	if len(want.List) != len(got.List) {
		return false
	}
	if len(want.Errors) != len(got.Errors) {
		return false
	}
	if !reflect.DeepEqual(want.Cite, got.Cite) {
		if len(want.Cite) != 0 && len(got.Cite) != 0 {
			return false
		}
	}
	// check error string equality
	for i := range want.Errors {
		if want.Errors[i].Error() != got.Errors[i].Error() {
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
		}},
	},
	{"*aa**bbb***c*dddd**ee***", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Italic, Beg: 0, End: 12},
				},
				Body: "*aa**bbb***c*dddd**ee***",
			},
		}},
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
		},
	}},
	{"*a`b*c`d*", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Italic, Beg: 0, End: 8},
					ast.Format{Kind: ast.Raw, Beg: 2, End: 6},
				},
				Body: "*a`b*c`d*",
			},
		},
	}},
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
		},
	}},
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
		},
	}},
	{"`[`]`", ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					ast.Format{Kind: ast.Raw, Beg: 0, End: 2},
				},
				Body: "`[`]`",
			},
		},
	}},
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
		},
	}},
}

func TestOverlap(t *testing.T) {
	litCfg := litter.Options{
		Compact:           true,
		StripPackageNames: false,
		HidePrivateFields: false,
		Separator:         " ",
	}
	for i, test := range overlapSmall {
		got := Parse(strings.NewReader(test.in))
		if !fileEquals(test.want, *got) {
			t.Errorf("case %d, in %q,\nwant %s, \ngot %s", i, test.in, litCfg.Sdump(test.want), litCfg.Sdump(*got))
		}
	}
}

func TestEscape(t *testing.T) {

}
func TestCombineListItem(t *testing.T) {

}
func TestCombineParagraph(t *testing.T) {

}
func TestUnicode(t *testing.T) {

}

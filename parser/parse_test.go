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

var combineListItem = []smallcase{
	{`- Single Line
- This text is
on multiple
lines.
- Ends with a single line.`, ast.File{
		List: []ast.Stmt{
			&ast.List{Items: []ast.ListItem{
				{Text: ast.Text{Body: " Single Line"}},
				{Text: ast.Text{Body: " This text is on multiple lines."}},
				{Text: ast.Text{Body: " Ends with a single line."}},
			}},
		}}, nil,
	},
	{`- Single Line
- This text is
on multiple
lines.

- Separate list.`, ast.File{
		List: []ast.Stmt{
			&ast.List{Items: []ast.ListItem{
				{Text: ast.Text{Body: " Single Line"}},
				{Text: ast.Text{Body: " This text is on multiple lines."}},
			}},
			&ast.List{Items: []ast.ListItem{
				{Text: ast.Text{Body: " Separate list."}},
			}},
		}}, nil,
	},
	{`- Single Line
- This text is
on multiple
lines.
-
- The previous item counts.`, ast.File{
		List: []ast.Stmt{
			&ast.List{Items: []ast.ListItem{
				{Text: ast.Text{Body: " Single Line"}},
				{Text: ast.Text{Body: " This text is on multiple lines."}},
				{Text: ast.Text{Body: ""}},
				{Text: ast.Text{Body: " The previous item counts."}},
			}},
		}}, nil,
	},
}

func TestCombineListItem(t *testing.T) {
	litCfg := litter.Options{
		Compact:           true,
		StripPackageNames: false,
		HidePrivateFields: false,
		Separator:         " ",
	}
	for i, test := range combineListItem {
		got, err := parser.Parse(strings.NewReader(test.in))
		if wes, es := fmt.Sprint(test.werr), fmt.Sprint(err); es != wes || !fileEquals(test.want, *got) {
			t.Errorf("case %d, in %q,\nwant %s,\ngot %s,\nwant err %s,\ngot err %s", i, test.in, litCfg.Sdump(test.want), litCfg.Sdump(*got), wes, es)
		}
	}
}

var combineParagraph = []smallcase{
	{`First line.
Second line.`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{Body: "First line.\nSecond line."},
		}}, nil,
	},
	{`
First line.
Second line.`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{Body: "First line.\nSecond line."},
		}}, nil,
	},
	{`First line.

Second line.`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{Body: "First line.\nSecond line."},
		}}, nil,
	},
	{`First line.
Second line.
- list item`, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{Body: "First line.\nSecond line."},
			&ast.List{Items: []ast.ListItem{
				{Text: ast.Text{Body: " list item"}},
			}},
		}}, nil,
	},
}

func TestCombineParagraph(t *testing.T) {
	litCfg := litter.Options{
		Compact:           true,
		StripPackageNames: false,
		HidePrivateFields: false,
		Separator:         " ",
	}
	for i, test := range combineParagraph {
		got, err := parser.Parse(strings.NewReader(test.in))
		if wes, es := fmt.Sprint(test.werr), fmt.Sprint(err); es != wes || !fileEquals(test.want, *got) {
			t.Errorf("case %d, in %q,\nwant %s,\ngot %s,\nwant err %s,\ngot err %s", i, test.in, litCfg.Sdump(test.want), litCfg.Sdump(*got), wes, es)
		}
	}
}

const (
	/*
		For reference (English):
		`This is the first line.
		This is the second line.
		Here is a [citation].
		*This text is italicized.*
		**This text is bolded.**
		***This text is both bolded and italicized.***
		_This is underlined._
		--This is struck through.--`+
		"\n`This is a raw string.`"
	*/

	telugu = `ఇది మొదటి పంక్తి.
ఇది రెండవ పంక్తి.
ఇది ఒక [సూచన].
*నేను ఇటాలిక్ చేస్తున్నాను.*
**నేను బోల్డ్ టెక్స్ట్ చేస్తున్నాను.**
***నేను బోల్డ్ మరియు ఇటాలిక్ చేస్తున్నాను.***
_ఇది అండర్లైన్._
--ఈ పంక్తి కొట్టి వేయబడి౦ది.--` +
		"\n`ఇది ముడి టెక్స్ట్.`"

	arabic = `هذا هو الخط الأول.
هذا هو الخط الثاني.
هنا هو ‪[‬المرجع‪]‬.
*هذا النص مائل.*
**هذا النص غامق.**
***هذا النص عريض ومائل.***
_هذا النص لديها خط تحتها._
--هذا النص لديها خط يمر عليه.--` +
		"\n`هذا هو النص الخام.`"

	hebrew = `זוהי השורה הראשונה.
זוהי השורה השנייה.
הנה ‪[‬התייחסות‪]‬.
*טקסט זה הוא נטוי.*
**טקסט זה מודגש.**
***טקסט זה הוא גם מודגש וגם נטוי.***
_לטקסט זה יש שורה מתחתיו._
--לטקסט זה קו דרכו.--` +
		"\n`זוהי מחרוזת גלם.`"

	chinese_simplified = `这是第一行。
这是第二行。
这里是一[个引用]。
*这个文本是斜体。*
**这个文本加粗。**
***本课文既是粗体和斜体。***
_这有下划线。_
--这个文本有删除线。--` +
		"\n`这是一个原始字符串。`"

		/* mathematical symbols and emojis*/

	pumping_lemma_esc = `(∀L ⊆ ∑\*)
	(regular(L) ⇒
	((∃p ≥ 1)((∀w ∈ L)((|w| ≥ p) ⇒
	((∃x,y,z ∈ ∑*)(w = xyz ∧ (|y| ≥ 1 ∧ |xy| ≤ p ∧ (∀n ≥ 0)(xyⁿz ∈ L))))))))`
	pumping_lemma = `(∀L ⊆ ∑*)
	(regular(L) ⇒
	((∃p ≥ 1)((∀w ∈ L)((|w| ≥ p) ⇒
	((∃x,y,z ∈ ∑*)(w = xyz ∧ (|y| ≥ 1 ∧ |xy| ≤ p ∧ (∀n ≥ 0)(xyⁿz ∈ L))))))))`

	emoji_u70 = `0001F600|😀,😁,😂,😃,😄,😅,😆,😇,😈,😉,😊,😋,😌,😍,😎,😏,
0001F610|😐,😑,😒,😓,😔,😕,😖,😗,😘,😙,😚,😛,😜,😝,😞,😟,
0001F620|😠,😡,😢,😣,😤,😥,😦,😧,😨,😩,😪,😫,😬,😭,😮,😯,
0001F630|😰,😱,😲,😳,😴,😵,😶,😷,😸,😹,😺,😻,😼,😽,😾,😿,
0001F640|🙀,🙁,🙂,🙃,🙄,🙅,🙆,🙇,🙈,🙉,🙊,🙋,🙌,🙍,🙎,🙏,
0001F680|🚀,🚁,🚂,🚃,🚄,🚅,🚆,🚇,🚈,🚉,🚊,🚋,🚌,🚍,🚎,🚏,
0001F690|🚐,🚑,🚒,🚓,🚔,🚕,🚖,🚗,🚘,🚙,🚚,🚛,🚜,🚝,🚞,🚟,
0001F6A0|🚠,🚡,🚢,🚣,🚤,🚥,🚦,🚧,🚨,🚩,🚪,🚫,🚬,🚭,🚮,🚯,
0001F6B0|🚰,🚱,🚲,🚳,🚴,🚵,🚶,🚷,🚸,🚹,🚺,🚻,🚼,🚽,🚾,🚿,
0001F6C0|🛀,🛁,🛂,🛃,🛄,🛅,🛆,🛇,🛈,🛉,🛊,🛋,🛌,🛍,🛎,🛏,
0001F6D0|🛐,🛑,🛒,🛓,🛔,🛕,🛖,🛗,🛘,🛙,🛚,🛛,🛜,🛝,🛞,🛟,
0001F6E0|🛠,🛡,🛢,🛣,🛤,🛥,🛦,🛧,🛨,🛩,🛪,🛫,🛬,🛭,🛮,🛯,
0001F6F0|🛰,🛱,🛲,🛳,🛴,🛵,🛶,🛷,🛸,🛹,🛺,🛻,🛼,🛽,🛾,🛿,
0001F300|🌀,🌁,🌂,🌃,🌄,🌅,🌆,🌇,🌈,🌉,🌊,🌋,🌌,🌍,🌎,🌏,
0001F310|🌐,🌑,🌒,🌓,🌔,🌕,🌖,🌗,🌘,🌙,🌚,🌛,🌜,🌝,🌞,🌟,
0001F320|🌠,🌡,🌢,🌣,🌤,🌥,🌦,🌧,🌨,🌩,🌪,🌫,🌬,🌭,🌮,🌯,
0001F330|🌰,🌱,🌲,🌳,🌴,🌵,🌶,🌷,🌸,🌹,🌺,🌻,🌼,🌽,🌾,🌿,
0001F340|🍀,🍁,🍂,🍃,🍄,🍅,🍆,🍇,🍈,🍉,🍊,🍋,🍌,🍍,🍎,🍏,
0001F350|🍐,🍑,🍒,🍓,🍔,🍕,🍖,🍗,🍘,🍙,🍚,🍛,🍜,🍝,🍞,🍟,
0001F360|🍠,🍡,🍢,🍣,🍤,🍥,🍦,🍧,🍨,🍩,🍪,🍫,🍬,🍭,🍮,🍯,
0001F370|🍰,🍱,🍲,🍳,🍴,🍵,🍶,🍷,🍸,🍹,🍺,🍻,🍼,🍽,🍾,🍿,
0001F380|🎀,🎁,🎂,🎃,🎄,🎅,🎆,🎇,🎈,🎉,🎊,🎋,🎌,🎍,🎎,🎏,
0001F390|🎐,🎑,🎒,🎓,🎔,🎕,🎖,🎗,🎘,🎙,🎚,🎛,🎜,🎝,🎞,🎟,
0001F3A0|🎠,🎡,🎢,🎣,🎤,🎥,🎦,🎧,🎨,🎩,🎪,🎫,🎬,🎭,🎮,🎯,
0001F3B0|🎰,🎱,🎲,🎳,🎴,🎵,🎶,🎷,🎸,🎹,🎺,🎻,🎼,🎽,🎾,🎿,
0001F3C0|🏀,🏁,🏂,🏃,🏄,🏅,🏆,🏇,🏈,🏉,🏊,🏋,🏌,🏍,🏎,🏏,
0001F3D0|🏐,🏑,🏒,🏓,🏔,🏕,🏖,🏗,🏘,🏙,🏚,🏛,🏜,🏝,🏞,🏟,
0001F3E0|🏠,🏡,🏢,🏣,🏤,🏥,🏦,🏧,🏨,🏩,🏪,🏫,🏬,🏭,🏮,🏯,
0001F3F0|🏰,🏱,🏲,🏳,🏴,🏵,🏶,🏷,🏸,🏹,🏺,🏻,🏼,🏽,🏾,🏿,
0001F400|🐀,🐁,🐂,🐃,🐄,🐅,🐆,🐇,🐈,🐉,🐊,🐋,🐌,🐍,🐎,🐏,
0001F410|🐐,🐑,🐒,🐓,🐔,🐕,🐖,🐗,🐘,🐙,🐚,🐛,🐜,🐝,🐞,🐟,
0001F420|🐠,🐡,🐢,🐣,🐤,🐥,🐦,🐧,🐨,🐩,🐪,🐫,🐬,🐭,🐮,🐯,
0001F430|🐰,🐱,🐲,🐳,🐴,🐵,🐶,🐷,🐸,🐹,🐺,🐻,🐼,🐽,🐾,🐿,
0001F440|👀,👁,👂,👃,👄,👅,👆,👇,👈,👉,👊,👋,👌,👍,👎,👏,
0001F450|👐,👑,👒,👓,👔,👕,👖,👗,👘,👙,👚,👛,👜,👝,👞,👟,
0001F460|👠,👡,👢,👣,👤,👥,👦,👧,👨,👩,👪,👫,👬,👭,👮,👯,
0001F470|👰,👱,👲,👳,👴,👵,👶,👷,👸,👹,👺,👻,👼,👽,👾,👿,
0001F480|💀,💁,💂,💃,💄,💅,💆,💇,💈,💉,💊,💋,💌,💍,💎,💏,
0001F490|💐,💑,💒,💓,💔,💕,💖,💗,💘,💙,💚,💛,💜,💝,💞,💟,
0001F4A0|💠,💡,💢,💣,💤,💥,💦,💧,💨,💩,💪,💫,💬,💭,💮,💯,
0001F4B0|💰,💱,💲,💳,💴,💵,💶,💷,💸,💹,💺,💻,💼,💽,💾,💿,
0001F4C0|📀,📁,📂,📃,📄,📅,📆,📇,📈,📉,📊,📋,📌,📍,📎,📏,
0001F4D0|📐,📑,📒,📓,📔,📕,📖,📗,📘,📙,📚,📛,📜,📝,📞,📟,
0001F4E0|📠,📡,📢,📣,📤,📥,📦,📧,📨,📩,📪,📫,📬,📭,📮,📯,
0001F4F0|📰,📱,📲,📳,📴,📵,📶,📷,📸,📹,📺,📻,📼,📽,📾,📿,
0001F500|🔀,🔁,🔂,🔃,🔄,🔅,🔆,🔇,🔈,🔉,🔊,🔋,🔌,🔍,🔎,🔏,
0001F510|🔐,🔑,🔒,🔓,🔔,🔕,🔖,🔗,🔘,🔙,🔚,🔛,🔜,🔝,🔞,🔟,
0001F520|🔠,🔡,🔢,🔣,🔤,🔥,🔦,🔧,🔨,🔩,🔪,🔫,🔬,🔭,🔮,🔯,
0001F530|🔰,🔱,🔲,🔳,🔴,🔵,🔶,🔷,🔸,🔹,🔺,🔻,🔼,🔽,🔾,🔿,
0001F540|🕀,🕁,🕂,🕃,🕄,🕅,🕆,🕇,🕈,🕉,🕊,🕋,🕌,🕍,🕎,🕏,
0001F550|🕐,🕑,🕒,🕓,🕔,🕕,🕖,🕗,🕘,🕙,🕚,🕛,🕜,🕝,🕞,🕟,
0001F560|🕠,🕡,🕢,🕣,🕤,🕥,🕦,🕧,🕨,🕩,🕪,🕫,🕬,🕭,🕮,🕯,
0001F570|🕰,🕱,🕲,🕳,🕴,🕵,🕶,🕷,🕸,🕹,🕺,🕻,🕼,🕽,🕾,🕿,
0001F580|🖀,🖁,🖂,🖃,🖄,🖅,🖆,🖇,🖈,🖉,🖊,🖋,🖌,🖍,🖎,🖏,
0001F590|🖐,🖑,🖒,🖓,🖔,🖕,🖖,🖗,🖘,🖙,🖚,🖛,🖜,🖝,🖞,🖟,
0001F5A0|🖠,🖡,🖢,🖣,🖤,🖥,🖦,🖧,🖨,🖩,🖪,🖫,🖬,🖭,🖮,🖯,
0001F5B0|🖰,🖱,🖲,🖳,🖴,🖵,🖶,🖷,🖸,🖹,🖺,🖻,🖼,🖽,🖾,🖿,
0001F5C0|🗀,🗁,🗂,🗃,🗄,🗅,🗆,🗇,🗈,🗉,🗊,🗋,🗌,🗍,🗎,🗏,
0001F5D0|🗐,🗑,🗒,🗓,🗔,🗕,🗖,🗗,🗘,🗙,🗚,🗛,🗜,🗝,🗞,🗟,
0001F5E0|🗠,🗡,🗢,🗣,🗤,🗥,🗦,🗧,🗨,🗩,🗪,🗫,🗬,🗭,🗮,🗯,
0001F5F0|🗰,🗱,🗲,🗳,🗴,🗵,🗶,🗷,🗸,🗹,🗺,🗻,🗼,🗽,🗾,🗿,`
)

var unicodeSmall = []smallcase{
	{telugu, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					{Kind: ast.Cite, Beg: 43, End: 48},
					{Kind: ast.Italic, Beg: 51, End: 78},
					{Kind: ast.Bold, Beg: 81, End: 117},
					{Kind: ast.BoldItalic, Beg: 121, End: 163},
					{Kind: ast.Underline, Beg: 165, End: 180},
					{Kind: ast.Strikethrough, Beg: 183, End: 211},
					{Kind: ast.Raw, Beg: 213, End: 232},
				}, // for now
				Body: telugu,
			}},
	}, nil},

	{arabic, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					{Kind: ast.Cite, Beg: 47, End: 56},
					{Kind: ast.Italic, Beg: 60, End: 75},
					{Kind: ast.Bold, Beg: 78, End: 94},
					{Kind: ast.BoldItalic, Beg: 98, End: 121},
					{Kind: ast.Underline, Beg: 123, End: 148},
					{Kind: ast.Strikethrough, Beg: 151, End: 180},
					{Kind: ast.Raw, Beg: 182, End: 201},
				}, // for now
				Body: arabic,
			}},
	}, nil},

	{hebrew, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					{Kind: ast.Cite, Beg: 44, End: 55},
					{Kind: ast.Italic, Beg: 59, End: 77},
					{Kind: ast.Bold, Beg: 80, End: 96},
					{Kind: ast.BoldItalic, Beg: 100, End: 133},
					{Kind: ast.Underline, Beg: 135, End: 160},
					{Kind: ast.Strikethrough, Beg: 163, End: 182},
					{Kind: ast.Raw, Beg: 184, End: 201},
				}, // for now
				Body: hebrew,
			}},
	}, nil},

	{chinese_simplified, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: []ast.Format{
					{Kind: ast.Cite, Beg: 18, End: 22},
					{Kind: ast.Italic, Beg: 25, End: 34},
					{Kind: ast.Bold, Beg: 37, End: 46},
					{Kind: ast.BoldItalic, Beg: 50, End: 64},
					{Kind: ast.Underline, Beg: 66, End: 73},
					{Kind: ast.Strikethrough, Beg: 76, End: 87},
					{Kind: ast.Raw, Beg: 89, End: 100},
				}, // for now
				Body: chinese_simplified,
			}},
	}, nil},

	{pumping_lemma_esc, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{
				Format: nil,
				Body:   pumping_lemma,
			}},
	}, nil},

	{emoji_u70, ast.File{
		List: []ast.Stmt{
			&ast.Paragraph{Body: emoji_u70}},
	}, nil},
}

func TestUnicode(t *testing.T) {
	litCfg := litter.Options{
		Compact:           true,
		StripPackageNames: false,
		HidePrivateFields: false,
		Separator:         " ",
	}
	for i, test := range unicodeSmall {
		got, err := parser.Parse(strings.NewReader(test.in))
		if wes, es := fmt.Sprint(test.werr), fmt.Sprint(err); es != wes || !fileEquals(test.want, *got) {
			t.Errorf("case %d, in %q,\nwant %s,\ngot %s,\nwant err %s,\ngot err %s", i, test.in, litCfg.Sdump(test.want), litCfg.Sdump(*got), wes, es)
		}
	}
}

package html

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/smasher164/mexdown/parse"
)

type smallcase struct {
	in   string
	want string
}

var escapeSmall = []smallcase{
	{"`n >= 3`", "<p><code>n &gt;= 3</code></p>"},
	{"*`n >= 3`*", "<p><em><code>n &gt;= 3</code></em></p>"},
	{"**`n >= 3`**", "<p><strong><code>n &gt;= 3</code></strong></p>"},
	{"**`n`**", "<p><strong><code>n</code></strong></p>"},
	{"A**`n >= 3`**B**`n`**C", "<p>A<strong><code>n &gt;= 3</code></strong>B<strong><code>n</code></strong>C</p>"},
}

func TestEscape(t *testing.T) {
	for i, test := range escapeSmall {
		f := parse.Parse(strings.NewReader(test.in))
		var buf bytes.Buffer
		io.Copy(&buf, &Genner{File: f})
		got := buf.String()
		if test.want != got {
			t.Errorf("case %d, in %q,\nwant %s, \ngot %s", i, test.in, test.want, got)
		}
	}
}

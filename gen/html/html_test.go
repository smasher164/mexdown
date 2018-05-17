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

// Tests for html.go
package html

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"akhil.cc/mexdown/parse"
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
		f, err := parse.Parse(strings.NewReader(test.in))
		if err != nil {
			t.Errorf("case %d, in %q,\nwant %s, \ngot error %s", i, test.in, test.want, err.Error())
			continue
		}
		var buf bytes.Buffer
		io.Copy(&buf, &Genner{File: f})
		got := buf.String()
		if test.want != got {
			t.Errorf("case %d, in %q,\nwant %s, \ngot %s", i, test.in, test.want, got)
		}
	}
}

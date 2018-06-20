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

// Examples for parse.go
package parser_test

import (
	"fmt"
	"strings"

	"akhil.cc/mexdown/ast"
	"akhil.cc/mexdown/parser"
)

func ExampleMustParse() {
	src := `# Favorite Hobbits
- Frodo
- Samwise
- Bilbo
- Merry
- Pippin
`
	file := parser.MustParse(strings.NewReader(src))
	for _, stmt := range file.List {
		switch t := stmt.(type) {
		case *ast.Header:
			fmt.Println(t.Text.Body)
		case *ast.List:
			for _, item := range t.Items {
				fmt.Println(item.Text.Body)
			}
		}
	}
	// Output:
	// Favorite Hobbits
	//  Frodo
	//  Samwise
	//  Bilbo
	//  Merry
	//  Pippin
}

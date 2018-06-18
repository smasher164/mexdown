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

// Examples for html.go
package html_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"akhil.cc/mexdown/gen/html"
	"akhil.cc/mexdown/parser"
)

func ExampleGen() {
	src := `# Heading 1
This is a paragraph.
*something something Gopher...*
`
	file := parser.MustParse(strings.NewReader(src))
	g := html.Gen(file)
	var out bytes.Buffer
	g.Stdout = &out

	if err := g.Run(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out.String())
	// Output:
	// <h1> Heading 1</h1><p>This is a paragraph.
	// <em>something something Gopher...</em></p>
}

func ExampleGenContext() {
	src := "The following is a directive:\n```" +
		`sh -c "for i in {1..5}; do echo $i; sleep 1; done"` +
		"\n```"
	file := parser.MustParse(strings.NewReader(src))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	g := html.GenContext(ctx, file)
	var out bytes.Buffer
	g.Stdout = &out

	if err := g.Run(); err != nil {
		// The 5 second sleep will be interrupted by the 2 second timeout.
	}
	fmt.Printf("%s\n", out.String())
}

func ExampleGenerator_Start() {
	src := "The following is a directive:\n" +
		"```sh -c \"for i in {1..5}; do echo $i; sleep 1; done\"\n" +
		"```"
	file := parser.MustParse(strings.NewReader(src))
	g := html.Gen(file)
	var out bytes.Buffer
	g.Stdout = &out

	if err := g.Start(); err != nil {
		log.Fatal(err)
	}
	log.Print("Waiting for generator to finish...")
	err := g.Wait()
	log.Printf("Generator finished with error: %v", err)
	fmt.Printf("%s\n", out.String())
}

func ExampleGenerator_Run() {
	src := "This document simply displays the date when compiled:\n" +
		"```date\n" +
		"```"
	file := parser.MustParse(strings.NewReader(src))
	g := html.Gen(file)
	var out bytes.Buffer
	g.Stdout = &out

	err := g.Run()
	log.Printf("Generator finished with error: %v", err)
	fmt.Printf("%s\n", out.String())
}

func ExampleGenerator_StdoutPipe() {
	src := `# Heading 1
This is a paragraph.
*something something Gopher...*
`
	file := parser.MustParse(strings.NewReader(src))
	g := html.Gen(file)
	stdout, err := g.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := g.Start(); err != nil {
		log.Fatal(err)
	}
	b, _ := ioutil.ReadAll(stdout)
	fmt.Printf("%s\n", b)

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}

func ExampleGenerator_StderrPipe() {
	src := "```sh -c \"echo standard output; echo standard error 1>&2\"\n" +
		"```"
	file := parser.MustParse(strings.NewReader(src))
	g := html.Gen(file)
	stderr, err := g.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := g.Start(); err != nil {
		log.Fatal(err)
	}
	b, _ := ioutil.ReadAll(stderr)
	fmt.Printf("Stderr: %s\n", b)

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}

func ExampleGenerator_Output() {
	src := `# Heading 1
This is a paragraph.
*something something Gopher...*
`
	file := parser.MustParse(strings.NewReader(src))
	b, err := html.Gen(file).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", b)
	// Output:
	// <h1> Heading 1</h1><p>This is a paragraph.
	// <em>something something Gopher...</em></p>
}

func ExampleGenerator_CombinedOutput() {
	src := "```sh -c \"echo standard output; echo standard error 1>&2\"\n" +
		"```"
	file := parser.MustParse(strings.NewReader(src))
	b, err := html.Gen(file).CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", b)
}

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

// This CLI utility runs a command listed below to run its
// corresponding output generator on a mexdown source file.
//
// Usage:
//   mexdown [command]
//
// Available Commands:
//   help        Help about any command
//   html        HTML output generator for mexdown source files
//
// Flags:
//   -h, --help   help for mexdown
//
// Use "mexdown [command] --help" for more information about a command.
package main

import (
	"context"
	"errors"
	"os"
	"time"

	"akhil.cc/mexdown/gen/html"
	"akhil.cc/mexdown/parser"
	"github.com/spf13/cobra"
)

func prefix(msg string, err error) error {
	return errors.New(msg + err.Error())
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "mexdown generator",
		Short: "output generation for mexdown source files",
		Long: `This CLI utility runs a command listed below to run its
corresponding output generator on a mexdown source file.`,
	}

	var outputfile string
	var timeout time.Duration
	prefixHTML := "(HTML) "
	htmlCmd := &cobra.Command{
		Use:   "html [input] [-o output]",
		Short: "HTML output generator for mexdown source files",
		Long: `This command takes a mexdown syntax tree and converts it to HTML.
Text inside raw string literals is automatically escaped. Overlapping
format tags in the source are converted into a tree structure.
Directives are parsed according to the Bourne shell's word-splitting rules.

If no input file is specified, input is read from
standard input. Similarly, if no output argument is
specified, output is written to standard output.`,
		Args: cobra.MaximumNArgs(1),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			src := os.Stdin
			var err error
			if len(args) != 0 {
				src, err = os.Open(args[0])
				if err != nil {
					return prefix(prefixHTML, err)
				}
			}
			defer src.Close()
			out := os.Stdout
			if len(outputfile) != 0 {
				out, err = os.Create(outputfile)
				if err != nil {
					return prefix(prefixHTML, err)
				}
			}
			defer out.Close()
			ast, err := parser.Parse(src)
			if err != nil {
				return prefix(prefixHTML, err)
			}
			ctx := context.Background()
			if timeout > -1 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
			g := html.GenContext(ctx, ast)
			g.Stdout = out
			g.Stderr = os.Stderr
			if err := g.Run(); err != nil {
				return prefix(prefixHTML, err)
			}
			return nil
		},
	}
	htmlCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err != nil {
			return prefix(prefixHTML, err)
		}
		return nil
	})
	// pflag includes the argument type when it unquotes its usage.
	// To prevent this behavior we prefix the usage with backquotes ``.
	htmlCmd.Flags().StringVarP(&outputfile, "output", "o", "", "``name of the output file")
	htmlCmd.Flags().DurationVarP(&timeout, "timeout", "t", -1, "``timeout used to halt generator for long-running commands")
	// Set string version of default value to be zero-value to prevent it from being printed by FlagUsages.
	htmlCmd.Flags().Lookup("timeout").DefValue = "0"

	rootCmd.AddCommand(htmlCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

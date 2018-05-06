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

package gen

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	sq "github.com/kballard/go-shellquote"
	"github.com/smasher164/mexdown/ast"
)

// Command holds the cancellation context and Stderr stream for a directive's executed process.
type Command struct {
	Ctx    context.Context
	Stderr io.Writer
}

// Gen executes the process and input in the directive node, waiting to
// finish writing its Stdout into w and Stderr into the command's Stderr.
// It returns any execution errors encountered during the process.
func (c *Command) Gen(dir *ast.Directive, w io.Writer) error {
	words, err := sq.Split(dir.Command)
	if err != nil {
		return err
	}
	if len(words) == 0 {
		return fmt.Errorf("No valid commands: '%q'", dir.Command)
	}
	var cmd *exec.Cmd
	if c.Ctx == nil {
		cmd = exec.Command(words[0], words[1:]...)
	} else {
		cmd = exec.CommandContext(c.Ctx, words[0], words[1:]...)
	}
	cmd.Stdin = strings.NewReader(dir.Raw)
	if w == nil && c.Stderr == nil {
		return fmt.Errorf("no output writer for directive %v", *dir)
	}
	if w != nil {
		cmd.Stdout = w
	}
	if c.Stderr != nil {
		cmd.Stderr = c.Stderr
	}
	return cmd.Run()
}

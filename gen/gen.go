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

type Command struct {
	Ctx    context.Context
	Stderr io.Writer
}

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

package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) ([]byte, error)
}

type Error struct {
	Name   string
	Args   []string
	Dir    string
	Output []byte
	Err    error
}

func (e *Error) Error() string {
	cmdline := e.Name
	if len(e.Args) > 0 {
		cmdline += " " + strings.Join(e.Args, " ")
	}
	if len(e.Output) > 0 {
		return fmt.Sprintf("%s: %v\nOutput: %s", cmdline, e.Err, e.Output)
	}
	return fmt.Sprintf("%s: %v", cmdline, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

type OS struct {
	Stream bool
}

func NewOS() *OS { return &OS{} }

func (r *OS) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	if r.Stream {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, &Error{Name: name, Args: args, Dir: dir, Err: err}
		}
		return nil, nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, &Error{Name: name, Args: args, Dir: dir, Output: output, Err: err}
	}
	return output, nil
}

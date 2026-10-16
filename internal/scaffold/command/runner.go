package command

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
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

type OS struct{}

func NewOS() *OS { return &OS{} }

const waitDelay = 5 * time.Second

func (r *OS) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.WaitDelay = waitDelay
	killTreeOnCancel(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, &Error{Name: name, Args: args, Dir: dir, Output: output, Err: err}
	}
	return output, nil
}

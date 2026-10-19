package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
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

var secretMarkers = []string{"PASSWORD", "SECRET", "TOKEN", "DSN", "KEY"}

func redactArg(arg string) string {
	name, _, found := strings.Cut(arg, "=")
	if !found {
		return arg
	}
	upper := strings.ToUpper(name)
	for _, marker := range secretMarkers {
		if strings.Contains(upper, marker) {
			return name + "=<redacted>"
		}
	}
	return arg
}

func (e *Error) Error() string {
	cmdline := e.Name
	if len(e.Args) > 0 {
		redacted := make([]string, len(e.Args))
		for i, arg := range e.Args {
			redacted[i] = redactArg(arg)
		}
		cmdline += " " + strings.Join(redacted, " ")
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

// colorForcing are the variables that make a tool print ANSI colour even when
// its output is not a terminal. The engine parses what it runs (pip and uv
// freezes, docker versions), so a user's FORCE_COLOR=1 must not reach them:
// "\x1b[1mdjango\x1b[0m==6.1.2" does not parse as a pin.
var colorForcing = []string{"FORCE_COLOR", "CLICOLOR_FORCE", "PY_COLORS"}

// plainOutputEnv is environ with colour forcing removed and NO_COLOR set, the
// convention (https://no-color.org) uv, pip, ruff and docker all honour.
func plainOutputEnv(environ []string) []string {
	env := make([]string, 0, len(environ)+1)
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		if slices.Contains(colorForcing, name) || name == "NO_COLOR" {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "NO_COLOR=1")
}

func (r *OS) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = plainOutputEnv(os.Environ())
	cmd.WaitDelay = waitDelay
	killTreeOnCancel(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, &Error{Name: name, Args: args, Dir: dir, Output: output, Err: err}
	}
	return output, nil
}

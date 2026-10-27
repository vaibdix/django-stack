package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/port"
)

type Runner = port.Runner

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

var colorForcing = []string{"FORCE_COLOR", "CLICOLOR_FORCE", "PY_COLORS"}

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
	return r.RunLines(ctx, dir, nil, name, args...)
}

func (r *OS) RunLines(ctx context.Context, dir string, line func(string), name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = plainOutputEnv(os.Environ())
	cmd.WaitDelay = waitDelay
	killTreeOnCancel(cmd)

	out := &lineWriter{line: line}
	cmd.Stdout, cmd.Stderr = out, out
	err := cmd.Run()
	out.flush()
	if err != nil {
		return out.all, &Error{Name: name, Args: args, Dir: dir, Output: out.all, Err: err}
	}
	return out.all, nil
}

type lineWriter struct {
	all     []byte
	pending int
	line    func(string)
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.all = append(w.all, p...)
	if w.line == nil {
		return len(p), nil
	}
	for {
		end := bytes.IndexByte(w.all[w.pending:], '\n')
		if end < 0 {
			return len(p), nil
		}
		w.line(strings.TrimRight(string(w.all[w.pending:w.pending+end]), "\r"))
		w.pending += end + 1
	}
}

func (w *lineWriter) flush() {
	if w.line != nil && w.pending < len(w.all) {
		w.line(strings.TrimRight(string(w.all[w.pending:]), "\r"))
		w.pending = len(w.all)
	}
}

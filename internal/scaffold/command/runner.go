// Package command centralises subprocess execution for the scaffolder.
//
// Every external tool the generator drives — python, uv, pip, django-admin,
// git, npm, docker — goes through a Runner. That gives three things the
// previous copy-pasted call sites did not have:
//
//   - a context on every invocation, so quitting the TUI actually stops the
//     child process instead of leaving pip writing into a half-built project;
//   - one consistent error that includes the captured output, instead of
//     twelve slightly different fmt.Errorf strings;
//   - a seam for tests, so scaffolding logic can be exercised without running
//     anything.
//
// It is named `command` rather than `exec` so callers can import os/exec
// alongside it without aliasing.
package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner executes external commands.
type Runner interface {
	// Run executes name with args in dir and returns the combined output.
	// A nil or empty dir runs in the caller's working directory.
	Run(ctx context.Context, dir, name string, args ...string) ([]byte, error)
}

// Error describes a subprocess that failed, retaining the output so the cause
// is visible without re-running anything.
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

// OS runs commands as real subprocesses.
type OS struct {
	// Stream, when true, connects the child's stdout/stderr to the parent's
	// instead of capturing them. pip and npm emit progress bars that look
	// broken when buffered and replayed, which is why the Windows install
	// paths historically streamed.
	Stream bool
}

// NewOS returns a Runner backed by real subprocesses.
func NewOS() *OS { return &OS{} }

// Run implements Runner.
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

// Look reports whether name is on PATH.
func Look(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

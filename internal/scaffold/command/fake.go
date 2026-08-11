package command

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Call records one invocation made through a Fake runner.
type Call struct {
	Dir  string
	Name string
	Args []string
}

// String renders the call as a shell-like line, which is what assertions and
// failure messages want to read.
func (c Call) String() string {
	if len(c.Args) == 0 {
		return c.Name
	}
	return c.Name + " " + strings.Join(c.Args, " ")
}

// Fake is a Runner that records calls instead of executing them.
//
// It lives in the non-test build so that packages outside this one can use it
// in their own tests; it has no dependency on the testing package.
type Fake struct {
	mu    sync.Mutex
	calls []Call

	// Responses maps a command line ("git init") to the output it should
	// return. Commands with no entry return empty output and no error.
	Responses map[string][]byte

	// Errors maps a command line to an error it should fail with.
	Errors map[string]error
}

// NewFake returns a Fake with initialised maps.
func NewFake() *Fake {
	return &Fake{
		Responses: make(map[string][]byte),
		Errors:    make(map[string]error),
	}
}

// Run implements Runner by recording the call and consulting Responses/Errors.
func (f *Fake) Run(_ context.Context, dir, name string, args ...string) ([]byte, error) {
	call := Call{Dir: dir, Name: name, Args: args}

	f.mu.Lock()
	f.calls = append(f.calls, call)
	responses, errors := f.Responses, f.Errors
	f.mu.Unlock()

	key := call.String()
	if err, ok := errors[key]; ok {
		return responses[key], &Error{
			Name: name, Args: args, Dir: dir, Output: responses[key], Err: err,
		}
	}
	return responses[key], nil
}

// Calls returns a copy of everything run so far.
func (f *Fake) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]Call, len(f.calls))
	copy(out, f.calls)
	return out
}

// CommandLines returns each recorded call rendered as a shell-like string.
func (f *Fake) CommandLines() []string {
	calls := f.Calls()
	lines := make([]string, len(calls))
	for i, c := range calls {
		lines[i] = c.String()
	}
	return lines
}

// Ran reports whether a command line was executed.
func (f *Fake) Ran(commandLine string) bool {
	return slices.Contains(f.CommandLines(), commandLine)
}

// FailWith makes the given command line fail with a generic error.
func (f *Fake) FailWith(commandLine, message string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Errors[commandLine] = fmt.Errorf("%s", message)
}

package command

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

type Call struct {
	Dir  string
	Name string
	Args []string
}

func (c Call) String() string {
	if len(c.Args) == 0 {
		return c.Name
	}
	return c.Name + " " + strings.Join(c.Args, " ")
}

type Fake struct {
	mu    sync.Mutex
	calls []Call

	Responses map[string][]byte

	Errors map[string]error
}

func NewFake() *Fake {
	return &Fake{
		Responses: make(map[string][]byte),
		Errors:    make(map[string]error),
	}
}

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

func (f *Fake) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]Call, len(f.calls))
	copy(out, f.calls)
	return out
}

func (f *Fake) CommandLines() []string {
	calls := f.Calls()
	lines := make([]string, len(calls))
	for i, c := range calls {
		lines[i] = c.String()
	}
	return lines
}

func (f *Fake) Ran(commandLine string) bool {
	return slices.Contains(f.CommandLines(), commandLine)
}

func (f *Fake) FailWith(commandLine, message string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Errors[commandLine] = fmt.Errorf("%s", message)
}

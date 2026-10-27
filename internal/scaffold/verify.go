package scaffold

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/render"
)

const verificationOutputLines = 40

type VerificationError struct {
	Check  string
	Output string
	Err    error
}

func (v *VerificationError) Error() string {
	if strings.TrimSpace(v.Output) == "" {
		return fmt.Sprintf("the project was generated, but its own %s fails: %v", v.Check, v.Err)
	}
	return fmt.Sprintf("the project was generated, but its own %s fails:\n%s", v.Check, v.Output)
}

func (v *VerificationError) Unwrap() error { return v.Err }

func (e *Env) verifyProject(ctx context.Context) error {
	checks := e.VerificationChecks()
	names := make([]string, 0, len(checks))
	for _, check := range checks {
		names = append(names, check.Name)
	}
	e.Step("Verifying: " + strings.Join(names, ", ") + "...")

	failures := make([]error, len(checks))
	var group sync.WaitGroup
	for _, lane := range verificationLanes(len(checks)) {
		group.Go(func() {
			for _, i := range lane {
				if failures[i] = e.runCheck(ctx, checks[i]); failures[i] != nil {
					return
				}
			}
		})
	}
	group.Wait()

	for _, failure := range failures {
		if failure != nil {
			return failure
		}
	}
	e.Step("Verified the project: " + strings.Join(names, ", ") + " pass.")
	return nil
}

func verificationLanes(n int) [][]int {
	if n <= 2 {
		lane := make([]int, n)
		for i := range lane {
			lane[i] = i
		}
		return [][]int{lane}
	}
	lanes := [][]int{{0, 1}}
	for i := 2; i < n; i++ {
		lanes = append(lanes, []int{i})
	}
	return lanes
}

func (e *Env) runCheck(ctx context.Context, check render.VerificationCheck) error {
	output, err := e.runIn(ctx, e.dir, pyenv.InterpreterPath(e.dir), check.Args...)
	if err != nil {
		return &VerificationError{Check: check.Name, Output: lastLines(string(output), verificationOutputLines), Err: err}
	}
	return nil
}

func lastLines(text string, n int) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

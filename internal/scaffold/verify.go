package scaffold

import (
	"fmt"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
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

type verificationCheck struct {
	name string
	args []string
}

func (e *Env) verificationChecks() []verificationCheck {
	test := []string{manageFile, "test", "--noinput"}
	if e.cfg.Has(config.MultiEnv) {
		test = append(test, "--settings="+e.cfg.ModuleName()+".settings.test")
	}

	checks := []verificationCheck{
		{name: "system check", args: []string{manageFile, "check"}},
		{name: "test suite", args: test},
	}
	if e.cfg.Has(config.Ruff) {
		checks = append(checks, verificationCheck{name: "ruff check", args: []string{"-m", "ruff", "check", "."}})
	}
	if e.cfg.Has(config.Mypy) {
		checks = append(checks, verificationCheck{name: "mypy check", args: []string{"-m", "mypy", "."}})
	}
	if e.cfg.Has(config.PreCommit) {
		checks = append(checks, verificationCheck{
			name: "pre-commit config",
			args: []string{"-m", "pre_commit", "validate-config", preCommitConfigFile},
		})
	}
	return checks
}

func (e *Env) verifyProject(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	python := pyenv.InterpreterPath(projectPath)
	checks := e.verificationChecks()
	for _, check := range checks {
		e.Step("Verifying: " + check.name + "...")
		output, err := e.runIn(ctx, projectPath, python, check.args...)
		if err != nil {
			return &VerificationError{Check: check.name, Output: lastLines(string(output), verificationOutputLines), Err: err}
		}
	}

	names := make([]string, 0, len(checks))
	for _, check := range checks {
		names = append(names, check.name)
	}
	e.Step("Verified the project: " + strings.Join(names, ", ") + " pass.")
	return nil
}

func lastLines(text string, n int) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

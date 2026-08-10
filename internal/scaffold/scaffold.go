// Package scaffold generates a Django project from a configuration.
//
// Nothing here imports Bubble Tea. The generator takes an immutable
// config.Project, writes files, runs external tools through a command.Runner,
// and reports what it did through a Reporter. That makes the whole engine
// runnable — and testable — without a terminal, which is what the previous
// arrangement (methods hanging off the TUI model) prevented.
//
// The Django-specific generation lives directly in this package rather than in
// a `djangoproj` subpackage: it is the bulk of the scaffolder, it all shares
// Env, and keeping it here means the leaf packages (pyenv, dbdocker, command,
// assets) never need to import back upward. The dependency graph is strictly
// one-way:
//
//	tui -> scaffold -> {pyenv, dbdocker, command, assets} -> config
package scaffold

import (
	"context"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

// Reporter receives progress as the scaffolder works.
//
// Step records something that happened. Progress additionally advances the
// progress bar, so it should be called once per planned step and no more — the
// two were conflated before, which is how the bar ended up exceeding 100%.
type Reporter interface {
	Step(message string)
	Progress(status string)
}

// NopReporter discards everything. Useful in tests.
type NopReporter struct{}

// Step implements Reporter by discarding the message.
func (NopReporter) Step(string) {}

// Progress implements Reporter by discarding the status.
func (NopReporter) Progress(string) {}

// RecordingReporter captures messages instead of displaying them, so a test can
// assert on what the scaffolder reported.
type RecordingReporter struct {
	Steps    []string
	Statuses []string
}

// Step implements Reporter by recording the message.
func (r *RecordingReporter) Step(message string) { r.Steps = append(r.Steps, message) }

// Progress implements Reporter by recording the status.
func (r *RecordingReporter) Progress(status string) { r.Statuses = append(r.Statuses, status) }

// Env is everything a scaffolding step needs: what to build, where to build it,
// how to run external tools, and where to report.
type Env struct {
	// Cfg is the user's choices. Steps read it; only the database steps write
	// back, to record the credentials of a container they started.
	Cfg config.Project

	// Root is the absolute path of the project directory being created.
	Root string

	// Runner executes external tools.
	Runner command.Runner

	// Reporter receives progress messages.
	Reporter Reporter

	// ctx bounds the whole generation run. It is set by Generate and read by
	// the per-command timeouts.
	//
	// Storing a context in a struct is normally discouraged, but Env is created
	// fresh for one generation and discarded, so it *is* the request scope.
	// The alternative — threading ctx through some thirty method signatures —
	// buys nothing here, and getting cancellation to reach every child process
	// is the entire point.
	ctx context.Context
}

// NewEnv returns an Env with sensible defaults for anything left unset.
func NewEnv(cfg config.Project, root string) *Env {
	return &Env{
		Cfg:      cfg,
		Root:     root,
		Runner:   command.NewOS(),
		Reporter: NopReporter{},
	}
}

// Step reports something the scaffolder did.
func (e *Env) Step(message string) {
	if e.Reporter != nil {
		e.Reporter.Step(message)
	}
}

// Progress advances the progress bar and reports the new status.
func (e *Env) Progress(status string) {
	if e.Reporter != nil {
		e.Reporter.Progress(status)
	}
}

// Run executes an external tool in the project directory.
func (e *Env) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return e.runner().Run(ctx, e.Root, name, args...)
}

// runIn executes an external tool in a specific directory.
func (e *Env) runIn(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	return e.runner().Run(ctx, dir, name, args...)
}

// Runner returns the configured runner, defaulting to real subprocesses.
func (e *Env) runner() command.Runner {
	if e.Runner == nil {
		return command.NewOS()
	}
	return e.Runner
}

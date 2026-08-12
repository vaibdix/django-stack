package scaffold

import (
	"context"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

type Reporter interface {
	Step(message string)
	Progress(status string)
}

type NopReporter struct{}

func (NopReporter) Step(string) {}

func (NopReporter) Progress(string) {}

type RecordingReporter struct {
	Steps    []string
	Statuses []string
}

func (r *RecordingReporter) Step(message string) { r.Steps = append(r.Steps, message) }

func (r *RecordingReporter) Progress(status string) { r.Statuses = append(r.Statuses, status) }

type Env struct {
	Cfg config.Project

	Root string

	Runner command.Runner

	Reporter Reporter

	Clock func() time.Time

	ctx context.Context

	deps dependencySet
}

func NewEnv(cfg config.Project, root string) *Env {
	return &Env{
		Cfg:      cfg,
		Root:     root,
		Runner:   command.NewOS(),
		Reporter: NopReporter{},
	}
}

func (e *Env) Step(message string) {
	if e.Reporter != nil {
		e.Reporter.Step(message)
	}
}

func (e *Env) Progress(status string) {
	if e.Reporter != nil {
		e.Reporter.Progress(status)
	}
}

func (e *Env) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return e.runner().Run(ctx, e.Root, name, args...)
}

func (e *Env) runIn(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	return e.runner().Run(ctx, dir, name, args...)
}

func (e *Env) runner() command.Runner {
	if e.Runner == nil {
		return command.NewOS()
	}
	return e.Runner
}

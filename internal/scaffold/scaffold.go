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

type Env struct {
	Cfg config.Project

	Root string

	Runner command.Runner

	Reporter Reporter

	Clock func() time.Time

	KeepOnFailure bool

	ctx context.Context

	deps dependencySet

	undo []func(context.Context) error

	frozen map[string]string
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

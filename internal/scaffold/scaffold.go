package scaffold

import (
	"context"
	"io"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold/command"
	"github.com/vaibdix/django-stack/internal/scaffold/pyenv"
)

const KeepFailedEnv = "DJANGO_STACK_KEEP_FAILED"

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

	toolchain pyenv.Toolchain

	undo []func(context.Context) error

	frozen map[string]string

	hashLookup func(ctx context.Context, project, version string) ([]string, error)

	nodeManagerLookup func() (NodeManager, bool)

	node NodeManager

	entropy io.Reader
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

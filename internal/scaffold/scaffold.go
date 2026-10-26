package scaffold

import (
	"context"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/report"
)

type Env struct {
	cfg config.Choices

	resolved config.Resolved

	root string

	dir string

	runner command.Runner

	reporter report.Reporter

	clock func() time.Time

	version string

	keepOnFailure bool

	checked bool

	packages dependencySet

	deps deps

	toolchain pyenv.Toolchain

	undo []func(context.Context) error

	written []string

	frozen map[string]string

	node NodeManager
}

type Options struct {
	Root          string
	Runner        command.Runner
	Reporter      report.Reporter
	Version       string
	Clock         func() time.Time
	KeepOnFailure bool
	Toolchain     pyenv.Toolchain
	NodeManager   NodeManager

	deps deps
}

func New(cfg config.Choices, resolved config.Resolved, opts Options) *Env {
	return &Env{
		cfg:           cfg,
		resolved:      resolved,
		root:          opts.Root,
		runner:        opts.Runner,
		reporter:      opts.Reporter,
		clock:         opts.Clock,
		version:       opts.Version,
		keepOnFailure: opts.KeepOnFailure,
		checked:       true,
		toolchain:     opts.Toolchain,
		node:          opts.NodeManager,
		deps:          opts.deps,
	}
}

func (e *Env) Step(message string) {
	if e.reporter != nil {
		e.reporter.Step(message)
	}
}

func (e *Env) Progress(status string) {
	if e.reporter != nil {
		e.reporter.Progress(status)
	}
}

func (e *Env) runIn(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	return e.commandRunner().Run(ctx, dir, name, args...)
}

func (e *Env) commandRunner() command.Runner {
	if e.runner == nil {
		return command.NewOS()
	}
	return e.runner
}

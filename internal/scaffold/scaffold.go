package scaffold

import (
	"context"
	"sync"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/report"
)

type project struct {
	cfg       config.Spec
	resolved  config.Resolved
	toolchain pyenv.Toolchain
	node      NodeManager
}

type Env struct {
	project

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

	mu sync.Mutex

	undo []func(context.Context) error

	written []string

	frozen map[string]string
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

func New(spec config.Spec, resolved config.Resolved, opts Options) *Env {
	return &Env{
		cfg:           spec,
		resolved:      resolved,
		toolchain:     opts.Toolchain,
		node:          opts.NodeManager,
		root:          opts.Root,
		runner:        opts.Runner,
		reporter:      opts.Reporter,
		clock:         opts.Clock,
		version:       opts.Version,
		keepOnFailure: opts.KeepOnFailure,
		checked:       true,
		deps:          opts.deps,
	}
}

func (e *Env) Step(message string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.reporter != nil {
		e.reporter.Step(message)
	}
}

func (e *Env) started(step, label string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.reporter != nil {
		e.reporter.Started(step, label)
	}
}

func (e *Env) finished(step, label string, took time.Duration, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.reporter != nil {
		e.reporter.Finished(step, label, took, err)
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

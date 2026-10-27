package scaffold

import (
	"context"
	"sync"
	"time"

	"github.com/vaibdix/django-stack/internal/adapter"
	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/event"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/render"
)

type Env struct {
	render.Project

	root string

	dir string

	target string

	targetExisted bool

	ports port.Ports

	events chan<- event.Event

	version string

	keepOnFailure bool

	checked bool

	packages render.DependencySet

	deps deps

	mu sync.Mutex

	undo []func(context.Context) error

	written []string

	frozen map[string]string
}

type Options struct {
	Root          string
	Ports         port.Ports
	Events        chan<- event.Event
	Version       string
	KeepOnFailure bool
	Toolchain     pyenv.Toolchain
	NodeManager   render.NodeManager

	deps deps
}

func New(spec config.Spec, resolved config.Resolved, opts Options) *Env {
	return &Env{
		Cfg:           spec,
		Resolved:      resolved,
		Toolchain:     opts.Toolchain,
		Node:          opts.NodeManager,
		root:          opts.Root,
		ports:         opts.Ports,
		events:        opts.Events,
		version:       opts.Version,
		keepOnFailure: opts.KeepOnFailure,
		checked:       true,
		deps:          opts.deps,
	}
}

func (e *Env) send(ev event.Event) {
	if e.events != nil {
		e.events <- ev
	}
}

func (e *Env) Step(message string) { e.send(event.Note{Text: message}) }

func (e *Env) started(effect, label string) {
	e.send(event.Started{Effect: effect, Label: label, At: e.now()})
}

func (e *Env) finished(effect, label string, took time.Duration, err error) {
	e.send(event.Finished{Effect: effect, Label: label, Took: took, Err: err})
}

func (e *Env) runIn(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	return e.commandRunner().Run(ctx, dir, name, args...)
}

func (e *Env) machine() port.Ports { return adapter.Complete(e.ports) }

func (e *Env) commandRunner() command.Runner { return streaming{inner: e.machine().Run, env: e} }

type streaming struct {
	inner command.Runner
	env   *Env
}

func (s streaming) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	lines, ok := s.inner.(port.LineRunner)
	effect, running := runningEffect(ctx)
	if !ok || !running || s.env.events == nil {
		return s.inner.Run(ctx, dir, name, args...)
	}
	return lines.RunLines(ctx, dir, func(line string) {
		s.env.send(event.Output{Effect: effect.ID, Line: line})
	}, name, args...)
}

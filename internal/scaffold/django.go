package scaffold

import (
	"context"
	"fmt"
	"runtime"

	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) createDjangoProject(ctx context.Context) error {
	if err := e.emitGroup(render.GroupProject); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Django project '%s' created.", e.Cfg.Name().String()))
	return nil
}

func (e *Env) reportPerformanceTip() {
	if e.Toolchain == pyenv.Pip && pyenv.IsWindows(runtime.GOOS) {
		e.Step(
			"Tip: Install 'uv' for faster package management: pip install uv")
	}
}

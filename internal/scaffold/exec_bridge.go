package scaffold

import (
	"context"
	"fmt"
	"time"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

const defaultCommandTimeout = 10 * time.Minute

func (e *Env) timeoutContext() (context.Context, context.CancelFunc) {
	parent := e.ctx
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, defaultCommandTimeout)
}

func (e *Env) createVirtualEnvironment(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	return pyenv.CreateVenv(ctx, e.runner(), e, projectPath, e.Cfg.SelectedPythonPath)
}

func (e *Env) installDjango(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	plan := e.packagePlan()
	if err := pyenv.InstallAll(ctx, e.runner(), e, projectPath, plan.all()...); err != nil {
		return err
	}

	e.deps.addRuntime(plan.runtime...)
	e.deps.addDev(plan.dev...)
	e.Step(fmt.Sprintf("Installed %d packages in one resolve.", len(plan.all())))
	return nil
}

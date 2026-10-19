package scaffold

import (
	"context"
	"fmt"
	"time"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

const defaultCommandTimeout = 10 * time.Minute

// context is the run's context: Generate sets it, and a step called on its
// own (tests, StartDevelopmentEnvironment) runs under a background one.
func (e *Env) context() context.Context {
	if e.ctx == nil {
		return context.Background()
	}
	return e.ctx
}

func (e *Env) timeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(e.context(), defaultCommandTimeout)
}

func (e *Env) createVirtualEnvironment(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	toolchain, err := pyenv.CreateVenv(ctx, e.runner(), e, projectPath, e.Cfg.SelectedPythonPath, e.toolchain)
	e.toolchain = toolchain
	return err
}

func (e *Env) installDjango(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	plan := e.packagePlan()
	if err := pyenv.InstallAll(ctx, e.runner(), e, e.toolchain, projectPath, plan.all()...); err != nil {
		return err
	}

	e.deps.addRuntime(plan.runtime...)
	e.deps.addDev(plan.dev...)
	e.Step(fmt.Sprintf("Installed %d packages in one resolve.", len(plan.all())))
	return nil
}

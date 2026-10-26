package scaffold

import (
	"context"
	"fmt"
	"time"

	"github.com/vaibdix/django-stack/internal/pyenv"
)

const defaultCommandTimeout = 10 * time.Minute

func (e *Env) createVirtualEnvironment(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	toolchain, err := pyenv.CreateVenv(ctx, e.commandRunner(), e, e.dir, e.resolved.PythonPath, e.toolchain)
	e.toolchain = toolchain
	return err
}

func (e *Env) installDjango(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	plan := e.packagePlan()
	if err := pyenv.InstallAll(ctx, e.commandRunner(), e, e.toolchain, e.dir, plan.all()...); err != nil {
		return err
	}

	e.packages.addRuntime(plan.runtime...)
	e.packages.addDev(plan.dev...)
	e.Step(fmt.Sprintf("Installed %d packages in one resolve.", len(plan.all())))
	return nil
}

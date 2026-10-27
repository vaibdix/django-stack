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

	toolchain, err := pyenv.CreateVenv(ctx, e.commandRunner(), e, e.dir, e.Resolved.PythonPath, e.Toolchain)
	e.Toolchain = toolchain
	return err
}

func (e *Env) installDjango(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	plan := e.PackagePlan()
	if err := pyenv.InstallAll(ctx, e.commandRunner(), e, e.Toolchain, e.dir, plan.All()...); err != nil {
		return err
	}

	e.packages.AddRuntime(plan.Runtime...)
	e.packages.AddDev(plan.Dev...)
	e.Step(fmt.Sprintf("Installed %d packages in one resolve.", len(plan.All())))
	return nil
}

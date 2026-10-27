package scaffold

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/vaibdix/django-stack/internal/pyenv"
)

const defaultCommandTimeout = 10 * time.Minute

func (e *Env) createVirtualEnvironment(ctx context.Context) error {
	toolchain, err := pyenv.CreateVenv(ctx, e.commandRunner(), e, e.dir, e.Resolved.PythonPath, e.Toolchain, e.staged())
	if err == nil && toolchain == pyenv.Pip && e.staged() {
		if err := e.machine().FS.RemoveAll(filepath.Join(e.dir, pyenv.VenvDir)); err != nil {
			return err
		}
		e.Step("A pip environment records its own path, so the project is built in place from here on.")
		if err := e.publish(); err != nil {
			return err
		}
		toolchain, err = pyenv.CreateVenv(ctx, e.commandRunner(), e, e.dir, e.Resolved.PythonPath, pyenv.Pip, false)
	}
	e.Toolchain = toolchain
	return err
}

func (e *Env) installDjango(ctx context.Context) error {
	plan := e.PackagePlan()
	if err := pyenv.InstallAll(ctx, e.commandRunner(), e, e.Toolchain, e.dir, plan.All()...); err != nil {
		return err
	}

	e.packages.AddRuntime(plan.Runtime...)
	e.packages.AddDev(plan.Dev...)
	e.Step(fmt.Sprintf("Installed %d packages in one resolve.", len(plan.All())))
	return nil
}

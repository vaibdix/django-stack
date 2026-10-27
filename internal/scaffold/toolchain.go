package scaffold

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
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
	if err := pyenv.InstallAll(ctx, e.commandRunner(), e, e.Toolchain, e.dir, append(e.resolveAtArgs(ctx), plan.All()...)...); err != nil {
		return err
	}

	e.packages.AddRuntime(plan.Runtime...)
	e.packages.AddDev(plan.Dev...)
	e.Step(fmt.Sprintf("Installed %d packages in one resolve.", len(plan.All())))
	return nil
}

func (e *Env) resolveAtArgs(ctx context.Context) []string {
	if e.Resolved.IndexDate.IsZero() {
		return nil
	}
	date := e.Resolved.IndexDate.UTC().Format(time.RFC3339)
	if e.Toolchain == pyenv.UV {
		return []string{"--exclude-newer", date}
	}
	help, err := e.runIn(ctx, e.dir, pyenv.InterpreterPath(e.dir), "-m", "pip", "install", "--help")
	if err == nil && strings.Contains(string(help), "--uploaded-prior-to") {
		return []string{"--uploaded-prior-to", date}
	}
	e.Step("This interpreter's pip cannot resolve at a fixed date, so its packages resolve against today's index.")
	return nil
}

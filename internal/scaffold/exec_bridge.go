package scaffold

import (
	"context"
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

func (e *Env) installPackages(projectPath, label string, packages ...string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	if err := pyenv.Install(ctx, e.runner(), e, projectPath, label, packages...); err != nil {
		return err
	}
	e.deps.addRuntime(packages...)
	return nil
}

func (e *Env) installDevPackages(projectPath, label string, packages ...string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	if err := pyenv.Install(ctx, e.runner(), e, projectPath, label, packages...); err != nil {
		return err
	}
	e.deps.addDev(packages...)
	return nil
}

func (e *Env) createVirtualEnvironment(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	return pyenv.CreateVenv(ctx, e.runner(), e, projectPath, e.Cfg.SelectedPythonPath)
}

func (e *Env) installDjango(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	extras := []string{PkgBrowserReload, PkgEnviron, PkgArgon2}
	if e.Cfg.DatabaseType.UsesPostgres() {
		extras = append(extras, PkgPsycopg)
	}

	if err := pyenv.InstallDjango(ctx, e.runner(), e, projectPath, e.Cfg.DjangoVersion, extras...); err != nil {
		return err
	}

	e.deps.addRuntime("django==" + e.Cfg.DjangoVersion)
	e.deps.addRuntime(extras...)

	e.Step("django-environ installed.")
	e.Step("argon2-cffi installed.")
	if e.Cfg.DatabaseType.UsesPostgres() {
		e.Step("psycopg installed.")
	}
	return nil
}

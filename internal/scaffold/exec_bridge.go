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

	return pyenv.Install(ctx, e.runner(), e, projectPath, label, packages...)
}

func (e *Env) createVirtualEnvironment(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	return pyenv.CreateVenv(ctx, e.runner(), e, projectPath, e.Cfg.SelectedPythonPath)
}

func (e *Env) installDjango(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	if err := pyenv.InstallDjango(ctx, e.runner(), e, projectPath, e.Cfg.DjangoVersion,
		PkgBrowserReload, PkgEnviron, PkgDotenv, PkgPsycopg, PkgArgon2,
	); err != nil {
		return err
	}

	e.Step("django-environ installed.")
	e.Step("python-dotenv installed.")
	e.Step("psycopg2-binary installed.")
	e.Step("argon2-cffi installed.")
	return nil
}

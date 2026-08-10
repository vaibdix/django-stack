package scaffold

import (
	"context"
	"time"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

// defaultCommandTimeout bounds any single external tool the scaffolder runs.
// Installing Django from a cold PyPI cache is the slowest realistic case.
const defaultCommandTimeout = 10 * time.Minute

// timeoutContext returns a context bounded by defaultCommandTimeout and by the
// lifetime of the generation run, so quitting mid-scaffold stops the current
// child process instead of leaving pip writing into a half-built project.
func (e *Env) timeoutContext() (context.Context, context.CancelFunc) {
	parent := e.ctx
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, defaultCommandTimeout)
}

// installPackages installs pip packages into the project's virtualenv.
func (e *Env) installPackages(projectPath, label string, packages ...string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	return pyenv.Install(ctx, e.runner(), e, projectPath, label, packages...)
}

// createVirtualEnvironment creates the project's virtual environment.
func (e *Env) createVirtualEnvironment(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	return pyenv.CreateVenv(ctx, e.runner(), e, projectPath, e.Cfg.SelectedPythonPath)
}

// installDjango installs Django plus the baseline runtime dependencies.
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

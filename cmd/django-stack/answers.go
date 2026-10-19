package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

// generateFromAnswers is the non-interactive run: the answers come from a
// file, progress goes to stderr as plain lines, and the outcome is the error
// main turns into an exit status.
func generateFromAnswers(path string, stderr io.Writer) error {
	answers, err := config.ReadAnswers(path)
	if err != nil {
		return err
	}
	project, err := answers.Project()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	run := command.NewOS()

	if err := choosePython(ctx, run, &project, answers.Python, time.Now()); err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	scaffold.Version = version
	env := &scaffold.Env{
		Cfg:           project,
		Root:          filepath.Join(wd, project.ProjectName),
		Runner:        run,
		Reporter:      lineReporter{w: stderr},
		KeepOnFailure: os.Getenv(scaffold.KeepFailedEnv) != "",
	}
	return env.Generate(ctx)
}

// choosePython runs the named interpreter to learn its version, then checks
// that the chosen Django release supports it, the same rule the interactive
// picker applies when it lists releases.
func choosePython(ctx context.Context, run command.Runner, project *config.Project, python string, today time.Time) error {
	options := pyenv.DiscoverFromCandidates(ctx, run, []string{python}, "")
	if len(options) == 0 {
		return fmt.Errorf("%q is not a usable Python %d.%d+ interpreter "+
			"(interpreters inside a .venv are not offered either)", python, pyenv.MinMajor, pyenv.MinMinor)
	}
	chosen := options[0]
	if !config.DjangoRunsOn(project.DjangoVersion, chosen.Major, chosen.Minor, today) {
		return fmt.Errorf("no supported Django series covers %s on Python %d.%d",
			project.DjangoVersion, chosen.Major, chosen.Minor)
	}
	project.SelectedPythonPath = chosen.Path
	project.SelectedPythonVersion = chosen.Version
	return nil
}

type lineReporter struct{ w io.Writer }

func (r lineReporter) Step(message string)    { _, _ = fmt.Fprintln(r.w, "  "+message) }
func (r lineReporter) Progress(status string) { _, _ = fmt.Fprintln(r.w, "==> "+status) }

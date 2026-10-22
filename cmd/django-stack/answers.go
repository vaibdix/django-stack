package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

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

var pythonVersionPattern = regexp.MustCompile(`^3\.(\d+)$`)

func choosePython(ctx context.Context, run command.Runner, project *config.Project, python string, today time.Time) error {
	chosen, err := findPython(ctx, run, python)
	if err != nil {
		return err
	}
	if !config.DjangoRunsOn(project.DjangoVersion, chosen.Major, chosen.Minor, today) {
		return fmt.Errorf("no supported Django series covers %s on Python %d.%d",
			project.DjangoVersion, chosen.Major, chosen.Minor)
	}
	project.SelectedPythonPath = chosen.Path
	project.SelectedPythonVersion = chosen.Version
	return nil
}

func findPython(ctx context.Context, run command.Runner, python string) (pyenv.Option, error) {
	match := pythonVersionPattern.FindStringSubmatch(python)
	if match == nil {
		options := pyenv.DiscoverFromCandidates(ctx, run, []string{python}, "")
		if len(options) == 0 {
			return pyenv.Option{}, fmt.Errorf("%q is not a usable Python %d.%d+ interpreter "+
				"(interpreters inside a .venv are not offered either)", python, pyenv.MinMajor, pyenv.MinMinor)
		}
		return options[0], nil
	}
	minor, err := strconv.Atoi(match[1])
	if err != nil {
		return pyenv.Option{}, err
	}
	options, _ := pyenv.Load(ctx, run)
	if chosen, ok := newestPatch(options, minor); ok {
		return chosen, nil
	}
	return pyenv.Option{}, fmt.Errorf("no usable Python %s interpreter was found on PATH, among uv's interpreters or in the usual install locations", python)
}

func newestPatch(options []pyenv.Option, minor int) (pyenv.Option, bool) {
	var best pyenv.Option
	found := false
	for _, option := range options {
		if option.Major == 3 && option.Minor == minor && (!found || option.Patch > best.Patch) {
			best, found = option, true
		}
	}
	return best, found
}

type lineReporter struct{ w io.Writer }

func (r lineReporter) Step(message string)    { _, _ = fmt.Fprintln(r.w, "  "+message) }
func (r lineReporter) Progress(status string) { _, _ = fmt.Fprintln(r.w, "==> "+status) }

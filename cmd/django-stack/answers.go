package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/releases"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

func generateFromAnswers(path string, stderr io.Writer, opts scaffold.Options) error {
	answers, err := config.ReadAnswers(path)
	if err != nil {
		return err
	}
	choices, err := answers.Choices()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		select {
		case sig := <-signals:
			cancel(signalError{sig})
		case <-ctx.Done():
		}
	}()
	run := command.NewOS()

	chosen, err := findPython(ctx, run, answers.Python)
	if err != nil {
		return err
	}
	spec, err := config.Parse(choices, config.Interpreter{
		Major: chosen.Major, Minor: chosen.Minor,
		Names: config.PythonNames{Stdlib: chosen.Stdlib, Keywords: chosen.Keywords},
	}, opts.Clock())
	if err != nil {
		return config.AnswerError(err, chosen.Version)
	}
	resolved := config.Resolved{PythonPath: chosen.Path, PythonVersion: chosen.Version}
	tools, err := scaffold.Preflight(ctx, spec, resolved, run)
	if err != nil {
		return err
	}
	catalog := releases.ForToolchain(tools.Toolchain, os.Getenv)
	if err := catalog.Resolve(ctx, chosen, choices.Django, opts.Clock()); err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if opts.Reporter == nil {
		opts.Reporter = lineReporter{w: stderr}
	}
	opts.Root, opts.Runner = filepath.Join(wd, choices.Name), run
	opts.Toolchain, opts.NodeManager = tools.Toolchain, tools.NodeManager
	err = scaffold.New(spec, resolved, opts).Generate(ctx)
	var signalled signalError
	if err != nil && errors.As(context.Cause(ctx), &signalled) {
		return fmt.Errorf("%w (%w)", err, signalled)
	}
	return err
}

type signalError struct{ signal os.Signal }

func (s signalError) Error() string { return "received " + s.signal.String() }

func (s signalError) exitCode() int {
	if number, ok := s.signal.(syscall.Signal); ok {
		return 128 + int(number)
	}
	return exitCancelled
}

var pythonVersionPattern = regexp.MustCompile(`^3\.(\d+)$`)

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
	options, _ := pyenv.Discover(ctx, run)
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

func (r lineReporter) Step(message string) { _, _ = fmt.Fprintln(r.w, "  "+message) }

func (r lineReporter) Started(_, label string) { _, _ = fmt.Fprintln(r.w, "==> "+label) }

func (r lineReporter) Finished(_, label string, took time.Duration, err error) {
	outcome := "done"
	if err != nil {
		outcome = "failed"
	}
	_, _ = fmt.Fprintf(r.w, "<== %s %s in %.2fs\n", label, outcome, took.Seconds())
}

type jsonReporter struct{ encoder *json.Encoder }

type jsonEvent struct {
	Event   string   `json:"event"`
	Step    string   `json:"step,omitempty"`
	Text    string   `json:"text,omitempty"`
	Seconds *float64 `json:"seconds,omitempty"`
	Error   string   `json:"error,omitempty"`
}

func newJSONReporter(w io.Writer) jsonReporter {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return jsonReporter{encoder: encoder}
}

func (r jsonReporter) Step(message string) {
	_ = r.encoder.Encode(jsonEvent{Event: "note", Text: message})
}

func (r jsonReporter) Started(step, label string) {
	_ = r.encoder.Encode(jsonEvent{Event: "started", Step: step, Text: label})
}

func (r jsonReporter) Finished(step, label string, took time.Duration, err error) {
	seconds := took.Seconds()
	event := jsonEvent{Event: "finished", Step: step, Text: label, Seconds: &seconds}
	if err != nil {
		event.Error = err.Error()
	}
	_ = r.encoder.Encode(event)
}

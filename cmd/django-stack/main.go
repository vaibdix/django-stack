package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/vaibdix/django-stack/internal/scaffold"
	"github.com/vaibdix/django-stack/internal/tui"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

const shortCommitLength = 7

type build struct {
	version string
	commit  string
	date    string
}

func (b build) withBuildInfo(info *debug.BuildInfo, ok bool) build {
	if !ok {
		return b
	}
	if b.version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		b.version = info.Main.Version
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if b.commit == "" {
				b.commit = setting.Value[:min(len(setting.Value), shortCommitLength)]
			}
		case "vcs.time":
			if b.date == "" {
				b.date = setting.Value
			}
		}
	}
	return b
}

type cli struct {
	build      build
	runProgram func(*tui.Model) error
}

func newCLI() cli {
	return cli{
		build:      build{version: version, commit: commit, date: date}.withBuildInfo(debug.ReadBuildInfo()),
		runProgram: startProgram,
	}
}

const keepFailedEnv = "DJANGO_STACK_KEEP_FAILED"

const (
	exitOK                 = 0
	exitFailed             = 1
	exitVerificationFailed = 2
	exitCancelled          = 130
)

func main() {
	err := newCLI().run(os.Args[1:], os.Stdout, os.Stderr)
	if code := exitCode(err); code != exitOK {
		fmt.Fprintln(os.Stderr, "django-stack:", err)
		os.Exit(code)
	}
}

func exitCode(err error) int {
	var verification *scaffold.VerificationError
	var signalled signalError
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return exitOK
	case errors.As(err, &signalled):
		return signalled.exitCode()
	case errors.As(err, &verification):
		return exitVerificationFailed
	case errors.Is(err, context.Canceled):
		return exitCancelled
	default:
		return exitFailed
	}
}

func (c cli) run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("django-stack", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		showVersion = fs.Bool("version", false, "print build version information and exit")
		doInstall   = fs.Bool("install", false, "copy this binary onto your PATH")
		answersPath = fs.String("config", "", "generate without the interactive setup, from a JSON answers `file`")
	)
	fs.Usage = func() { printUsage(stderr, fs) }

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() > 0 {
		printUsage(stderr, fs)
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}

	engine := scaffold.Options{Version: c.build.version, Clock: time.Now, KeepOnFailure: os.Getenv(keepFailedEnv) != ""}
	switch {
	case *showVersion:
		c.build.print(stdout)
		return nil
	case *doInstall:
		return install(stdout)
	case *answersPath != "":
		return generateFromAnswers(*answersPath, stderr, engine)
	}

	return c.runTUI(engine)
}

func startProgram(model *tui.Model) error {
	program := tea.NewProgram(model)
	model.SetProgram(program)
	_, err := program.Run()
	return err
}

func (c cli) runTUI(engine scaffold.Options) error {
	model := tui.NewModel(engine)
	if err := c.runProgram(model); err != nil {
		return fmt.Errorf("failed to run the setup interface: %w", err)
	}
	return model.Err()
}

func printUsage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprint(w, `django-stack — a TUI-first Django project generator.

Usage:
  django-stack [flags]

Without flags it starts the interactive setup, which walks through project
name, Python interpreter, Django version, database choice, and optional
project features.

Exit status: 0 when the project was created, 1 when setup failed (the partial
project is removed), 2 when the project was created but fails its own checks
(it is kept), 130 when setup was interrupted, 143 when it was terminated.

Flags:
`)
	fs.PrintDefaults()
}

func (b build) print(w io.Writer) {
	_, _ = fmt.Fprintf(w, "django-stack %s\n", b.version)
	if b.commit != "" {
		_, _ = fmt.Fprintf(w, "commit: %s\n", b.commit)
	}
	if b.date != "" {
		_, _ = fmt.Fprintf(w, "date: %s\n", b.date)
	}
}

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vaibdix/django-cli/internal/scaffold"
	"github.com/vaibdix/django-cli/internal/tui"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

var readBuildInfo = debug.ReadBuildInfo

const shortCommitLength = 7

func fillVersionFromBuildInfo() {
	info, ok := readBuildInfo()
	if !ok {
		return
	}
	if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if commit == "" {
				commit = setting.Value[:min(len(setting.Value), shortCommitLength)]
			}
		case "vcs.time":
			if date == "" {
				date = setting.Value
			}
		}
	}
}

// Exit statuses. A project that was generated but fails its own checks is
// kept on disk, so it gets its own status rather than the generic failure.
const (
	exitOK                 = 0
	exitFailed             = 1
	exitVerificationFailed = 2
	exitCancelled          = 130
)

func main() {
	fillVersionFromBuildInfo()

	err := run(os.Args[1:], os.Stdout, os.Stderr)
	if code := exitCode(err); code != exitOK {
		// Written after the TUI has left the alternate screen, so the error
		// is still on the terminal once the program has exited.
		fmt.Fprintln(os.Stderr, "django-stack:", err)
		os.Exit(code)
	}
}

func exitCode(err error) int {
	var verification *scaffold.VerificationError
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return exitOK
	case errors.As(err, &verification):
		return exitVerificationFailed
	case errors.Is(err, context.Canceled):
		return exitCancelled
	default:
		return exitFailed
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("django-stack", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		showVersion = fs.Bool("version", false, "print build version information and exit")
		doInstall   = fs.Bool("install", false, "copy this binary onto your PATH")
	)
	fs.Usage = func() { printUsage(stderr, fs) }

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() > 0 {
		printUsage(stderr, fs)
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}

	switch {
	case *showVersion:
		printVersion(stdout)
		return nil
	case *doInstall:
		return install(stdout)
	}

	return runTUI()
}

// runProgram runs the interactive setup to completion. It is a seam so tests
// can stand in for a terminal.
var runProgram = func(model *tui.Model) error {
	program := tea.NewProgram(model, tea.WithAltScreen())
	model.SetProgram(program)
	_, err := program.Run()
	return err
}

func runTUI() error {
	scaffold.Version = version

	model := tui.NewModel()
	if err := runProgram(model); err != nil {
		return fmt.Errorf("failed to run the setup interface: %w", err)
	}
	return model.Err()
}

func printUsage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprint(w, `django-stack — a TUI-first Django project generator.

Usage:
  django-stack              Start the interactive setup
  django-stack -version     Show build version information
  django-stack -install     Install the binary into your local PATH
  django-stack -h           Show this help

The setup flow walks through project name, Python interpreter, Django version,
database choice, and optional project features.

Exit status: 0 when the project was created, 1 when setup failed (the partial
project is removed), 2 when the project was created but fails its own checks
(it is kept), 130 when setup was cancelled.

Flags:
`)
	fs.PrintDefaults()
}

func printVersion(w io.Writer) {
	_, _ = fmt.Fprintf(w, "django-stack %s\n", version)
	if commit != "" {
		_, _ = fmt.Fprintf(w, "commit: %s\n", commit)
	}
	if date != "" {
		_, _ = fmt.Fprintf(w, "date: %s\n", date)
	}
}

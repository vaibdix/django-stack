package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vaibdix/django-cli/internal/tui"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {

		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "django-stack:", err)
		os.Exit(1)
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

func runTUI() error {
	model := tui.NewModel()
	program := tea.NewProgram(model, tea.WithAltScreen())
	model.SetProgram(program)

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run the setup interface: %w", err)
	}
	return nil
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

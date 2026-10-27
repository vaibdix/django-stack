package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const referencePython = "3.13"

func runReference(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("reference", flag.ContinueOnError)
	dir := flags.String("dir", "reference", "the reference project written by make reference")
	keep := flags.Bool("keep", false, "keep the checked copy and print where it is")
	if err := flags.Parse(args); err != nil {
		return err
	}

	work, err := os.MkdirTemp("", "django-stack-reference-")
	if err != nil {
		return err
	}
	if *keep {
		_, _ = fmt.Fprintln(stdout, "checking a copy at", work)
	} else {
		defer func() { _ = os.RemoveAll(work) }()
	}
	if err := os.CopyFS(work, os.DirFS(*dir)); err != nil {
		return fmt.Errorf("copying %s: %w", *dir, err)
	}
	example, err := os.ReadFile(filepath.Join(work, ".env.example"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(work, ".env"), example, 0o600); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	if err := referenceStep(ctx, work, stdout, "uv sync", "uv", "sync", "--quiet", "--python", referencePython); err != nil {
		return err
	}
	templates, err := templateDirs(work)
	if err != nil {
		return err
	}
	var failures []error
	for _, step := range [][]string{
		{"ruff format", "ruff", "format", "--check", "."},
		{"ruff check", "ruff", "check", "."},
		{"mypy --strict", "mypy", "."},
		slices.Concat([]string{"djlint", "djlint", "--lint", "--check", "--profile=django"}, templates),
		{"manage.py check", "python", "manage.py", "check"},
		{"makemigrations --check", "python", "manage.py", "makemigrations", "--check", "--dry-run"},
		{"manage.py test", "python", "manage.py", "test"},
	} {
		run := slices.Concat([]string{"uv", "run", "--no-sync"}, step[1:])
		failures = append(failures, referenceStep(ctx, work, stdout, step[0], run[0], run[1:]...))
	}
	return errors.Join(failures...)
}

func referenceStep(ctx context.Context, dir string, stdout io.Writer, label, name string, args ...string) error {
	started := time.Now()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool {
		return strings.HasPrefix(kv, "VIRTUAL_ENV=")
	})
	output, err := cmd.CombinedOutput()
	took := time.Since(started).Round(100 * time.Millisecond)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "FAIL %-24s %s\n", label, took)
		return fmt.Errorf("%s: %w\n%s", label, err, lastLines(string(output), 40))
	}
	_, _ = fmt.Fprintf(stdout, "ok   %-24s %s\n", label, took)
	return nil
}

func templateDirs(root string) ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		switch d.Name() {
		case ".venv", "node_modules", ".git":
			return filepath.SkipDir
		case "templates":
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			dirs = append(dirs, rel)
			return filepath.SkipDir
		}
		return nil
	})
	return dirs, err
}

package scaffold

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

var joinPath = filepath.Join

const rollbackTimeout = 2 * time.Minute

func (e *Env) Generate(ctx context.Context) (err error) {
	e.ctx = ctx

	if err := e.validate(); err != nil {
		return err
	}

	projectPath, err := e.prepareProjectDirectory()
	if err != nil {
		return err
	}

	defer func() { err = e.afterRun(ctx, projectPath, err) }()

	e.toolchain = pyenv.DetectToolchain(ctx, e.runner())
	e.Step("Package manager: " + e.toolchain.String())

	for _, s := range plan() {
		if !s.applies(e.Cfg) {
			continue
		}

		if ctx.Err() != nil {
			return fmt.Errorf("run cancelled before %s: %w", s.name, ctx.Err())
		}

		e.Progress(s.label)

		if err := s.run(e, projectPath); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("run cancelled during %s: %w", s.name, ctx.Err())
			}
			return fmt.Errorf("%s: %w", s.name, err)
		}
	}

	e.reportPerformanceTip()
	e.Step("Django project setup complete!")
	return nil
}

func (e *Env) afterRun(ctx context.Context, projectPath string, err error) error {
	var verification *VerificationError
	if err == nil || errors.As(err, &verification) {
		return err
	}
	return e.rollback(ctx, projectPath, err)
}

func (e *Env) onRollback(undo func(context.Context) error) {
	e.undo = append(e.undo, undo)
}

func (e *Env) rollback(ctx context.Context, projectPath string, cause error) error {
	if e.KeepOnFailure {
		return fmt.Errorf("%w\n\nThe partial project was kept at %s", cause, projectPath)
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
	defer cancel()

	var failures []error
	for i := len(e.undo) - 1; i >= 0; i-- {
		if err := e.undo[i](ctx); err != nil {
			failures = append(failures, err)
		}
	}
	e.undo = nil

	if len(failures) > 0 {
		return fmt.Errorf("%w\n\nCleaning up the partial project also failed: %w", cause, errors.Join(failures...))
	}
	return fmt.Errorf("%w\n\nThe partial project was removed, so the same name can be used again", cause)
}

func (e *Env) validate() error {
	if e.Cfg.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if err := e.Cfg.ValidateProjectName(); err != nil {
		return fmt.Errorf("invalid project name '%s': %w", e.Cfg.ProjectName, err)
	}
	if err := e.Cfg.ValidateAppName(); err != nil {
		return fmt.Errorf("invalid app name '%s': %w", e.Cfg.AppName, err)
	}
	if e.Cfg.SelectedPythonPath == "" {
		return errors.New("no Python interpreter was selected")
	}
	if e.Cfg.DjangoVersion == "" {
		return errors.New("no Django version was selected")
	}
	if e.Cfg.SetupTailwind {
		if _, ok := e.findNodeManager(); !ok {
			// Without a node package manager the CSS is never compiled, and
			// every page then references a stylesheet that does not exist.
			return errors.New("the Tailwind stacks need Node.js with pnpm or npm on PATH to compile their CSS; " +
				"install Node.js, or choose a stack without Tailwind")
		}
	}
	return nil
}

func (e *Env) findNodeManager() (NodeManager, bool) {
	if e.nodeManagerLookup != nil {
		return e.nodeManagerLookup()
	}
	return detectNodeManager()
}

func (e *Env) prepareProjectDirectory() (string, error) {
	projectPath := e.Root
	if projectPath == "" {
		projectPath = e.Cfg.ProjectName
	}
	if !filepath.IsAbs(projectPath) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		projectPath = filepath.Join(wd, e.Cfg.ProjectName)
	}

	if err := checkDirectoryIsUsable(projectPath); err != nil {
		return "", err
	}

	_, statErr := os.Stat(projectPath)
	existed := statErr == nil

	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %w", err)
	}
	e.onRollback(func(context.Context) error { return removeProjectDirectory(projectPath, existed) })
	e.Step(fmt.Sprintf("Project directory created: %s", projectPath))

	return projectPath, nil
}

func checkDirectoryIsUsable(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to inspect project directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("project path exists and is not a directory: %s", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to inspect project directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("project directory already exists and is not empty: %s", path)
	}
	return nil
}

func (e *Env) writeDevStartScript(projectPath string) error {
	for _, windows := range []bool{false, true} {
		name := devStartScriptNameFor(windows)
		mode := os.FileMode(0755)
		if windows {
			mode = 0644
		}
		if err := os.WriteFile(filepath.Join(projectPath, name), []byte(generateStartScript(windows, e.Cfg.SetupTailwind)), mode); err != nil {
			e.Step("Warning: Could not create " + name + ": " + err.Error())
			continue
		}
		e.Step("Created: " + name)
	}
	return nil
}

func removeProjectDirectory(projectPath string, existed bool) error {
	if !existed {
		return os.RemoveAll(projectPath)
	}
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(projectPath, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

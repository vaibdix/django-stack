package scaffold

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/releases"
)

var joinPath = filepath.Join

const rollbackTimeout = 2 * time.Minute

func (e *Env) Generate(ctx context.Context) (err error) {
	e.ctx = ctx

	if !e.checked {
		tools, err := preflight(ctx, e.cfg, e.commandRunner(), e.now(), func(ctx context.Context, _ command.Runner) (NodeManager, bool) {
			return e.findNodeManager(ctx)
		})
		if err != nil {
			return err
		}
		e.toolchain, e.node = tools.Toolchain, tools.NodeManager
	}

	projectPath, err := e.prepareProjectDirectory()
	if err != nil {
		return err
	}

	defer func() { err = e.afterRun(ctx, projectPath, err) }()

	e.Step("Package manager: " + e.toolchain.String())

	for _, s := range plan() {
		if !s.applies(e.cfg) {
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
	if e.keepOnFailure {
		return fmt.Errorf("%w\n\nThe partial project was kept at %s", cause, projectPath)
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
	defer cancel()

	var failures []error
	for _, undo := range slices.Backward(e.undo) {
		if err := undo(ctx); err != nil {
			failures = append(failures, err)
		}
	}
	e.undo = nil

	if len(failures) > 0 {
		return fmt.Errorf("%w\n\nCleaning up the partial project also failed: %w", cause, errors.Join(failures...))
	}
	return fmt.Errorf("%w\n\nThe partial project was removed, so the same name can be used again", cause)
}

type Tools struct {
	Toolchain   pyenv.Toolchain
	NodeManager NodeManager
}

func Preflight(ctx context.Context, cfg config.Project, run command.Runner, today time.Time) (Tools, error) {
	return preflight(ctx, cfg, run, today, detectNodeManager)
}

func preflight(ctx context.Context, cfg config.Project, run command.Runner, today time.Time,
	findNodeManager func(context.Context, command.Runner) (NodeManager, bool),
) (Tools, error) {
	if cfg.SelectedPythonPath == "" {
		return Tools{}, errors.New("no Python interpreter was selected")
	}
	if cfg.DjangoVersion == "" {
		return Tools{}, errors.New("no Django version was selected")
	}
	var names config.PythonNames
	if probed, ok := pyenv.Probe(ctx, run, cfg.SelectedPythonPath); ok {
		names = config.PythonNames{Stdlib: probed.Stdlib, Keywords: probed.Keywords}
	}
	if err := cfg.ValidateProjectName(names); err != nil {
		return Tools{}, fmt.Errorf("invalid project name '%s': %w", cfg.ProjectName, err)
	}
	if err := cfg.ValidateAppName(names); err != nil {
		return Tools{}, fmt.Errorf("invalid app name '%s': %w", cfg.AppName, err)
	}
	if major, minor, _, _, ok := pyenv.ParseDetailedVersionOutput(cfg.SelectedPythonVersion); ok &&
		!config.DjangoRunsOn(cfg.DjangoVersion, major, minor, today) {
		return Tools{}, fmt.Errorf("the selected Django release %s does not support Python %d.%d; choose another release or interpreter",
			cfg.DjangoVersion, major, minor)
	}
	if cfg.Has(config.Git) && !pyenv.CommandWorks(ctx, run, "git") {
		return Tools{}, errors.New("git is not on PATH; install Git, or leave out the git and pre-commit add-ons")
	}
	if cfg.DatabaseType.UsesDocker() {
		if err := dbdocker.Ready(ctx, run, cfg.ProjectName, cfg.DatabaseType); err != nil {
			return Tools{}, err
		}
	}
	var tools Tools
	if cfg.UsesTailwind() {
		manager, ok := findNodeManager(ctx, run)
		if !ok {
			return Tools{}, errors.New("the Tailwind stacks need Node.js with pnpm or npm on PATH to compile their CSS; " +
				"install Node.js, or choose a stack without Tailwind")
		}
		if err := requireNode(ctx, run); err != nil {
			return Tools{}, err
		}
		tools.NodeManager = manager
	}
	tools.Toolchain = releases.DetectToolchain(ctx, run)
	return tools, nil
}

func requireNode(ctx context.Context, run command.Runner) error {
	output, err := run.Run(ctx, "", "node", "--version")
	if err != nil {
		return fmt.Errorf("the Tailwind stacks need Node.js %d or newer, and `node --version` failed: %w", tailwindMinNode, err)
	}
	major, err := strconv.Atoi(strings.SplitN(strings.TrimPrefix(strings.TrimSpace(string(output)), "v"), ".", 2)[0])
	if err != nil {
		return fmt.Errorf("could not read the Node.js version from %q", strings.TrimSpace(string(output)))
	}
	if major < tailwindMinNode {
		return fmt.Errorf("the Tailwind CSS %s build needs Node.js %d or newer; this machine has %s", tailwindVersion, tailwindMinNode, strings.TrimSpace(string(output)))
	}
	return nil
}

func (e *Env) findNodeManager(ctx context.Context) (NodeManager, bool) {
	if e.nodeManagerLookup != nil {
		return e.nodeManagerLookup()
	}
	return detectNodeManager(ctx, e.commandRunner())
}

func (e *Env) projectPath() (string, error) {
	path := e.root
	if path == "" {
		path = e.cfg.ProjectName
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve the project directory %s: %w", path, err)
	}
	return absolute, nil
}

func (e *Env) prepareProjectDirectory() (string, error) {
	projectPath, err := e.projectPath()
	if err != nil {
		return "", err
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
	e.Step("Project directory created: " + projectPath)

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
	for _, goos := range []string{"linux", pyenv.Windows} {
		name := devStartScriptNameFor(goos)
		mode := os.FileMode(0755)
		if pyenv.IsWindows(goos) {
			mode = 0644
		}
		if err := os.WriteFile(filepath.Join(projectPath, name), []byte(generateStartScript(goos, e.cfg.UsesTailwind())), mode); err != nil {
			return fmt.Errorf("failed to create %s: %w", name, err)
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

package scaffold

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/adapter"
	"github.com/vaibdix/django-stack/internal/catalog"
	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/graph"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/releases"
)

const rollbackTimeout = 2 * time.Minute

func (e *Env) Generate(ctx context.Context) (err error) {
	if !e.checked {
		spec, err := probeAndParse(ctx, e.cfg.Choices(), e.resolved, e.commandRunner(), e.now())
		if err != nil {
			return err
		}
		tools, err := preflight(ctx, spec, e.resolved, e.machine(), e.dependencies())
		if err != nil {
			return err
		}
		e.toolchain, e.node = tools.Toolchain, tools.NodeManager
	}
	if e.resolved.Generated.IsZero() {
		e.resolved.Generated = e.now()
	}

	if err := e.resolveSecrets(); err != nil {
		return err
	}

	projectPath, err := e.prepareProjectDirectory()
	if err != nil {
		return err
	}
	e.dir = projectPath

	defer func() { err = e.afterRun(ctx, projectPath, err) }()

	e.Step("Package manager: " + e.toolchain.String())

	label := labels()
	err = graph.Run(ctx, e, nodes(e.cfg), graph.Hooks{
		Started:  func(name string) { e.started(name, label[name]) },
		Finished: func(name string, took time.Duration, err error) { e.finished(name, label[name], took, err) },
		Clock:    e.now,
	})
	if failure, ok := errors.AsType[*graph.Failure](err); ok {
		switch {
		case !failure.Ran:
			return fmt.Errorf("run cancelled before %s: %w", failure.Node, failure.Err)
		case ctx.Err() != nil:
			return fmt.Errorf("run cancelled during %s: %w", failure.Node, ctx.Err())
		default:
			return fmt.Errorf("%s: %w", failure.Node, failure.Err)
		}
	}
	if err != nil {
		return err
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
	e.mu.Lock()
	defer e.mu.Unlock()
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

func probeAndParse(ctx context.Context, cfg config.Choices, resolved config.Resolved, run command.Runner, today time.Time) (config.Spec, error) {
	var python config.Interpreter
	if probed, ok := pyenv.Probe(ctx, run, resolved.PythonPath); ok {
		python.Names = config.PythonNames{Stdlib: probed.Stdlib, Keywords: probed.Keywords}
	}
	if major, minor, _, _, ok := pyenv.ParseDetailedVersionOutput(resolved.PythonVersion); ok {
		python.Major, python.Minor = major, minor
	}
	spec, _, err := config.Parse(cfg, python, today)
	return spec, err
}

func Preflight(ctx context.Context, spec config.Spec, resolved config.Resolved, ports port.Ports) (Tools, error) {
	return preflight(ctx, spec, resolved, adapter.Complete(ports), deps{}.withDefaults())
}

func preflight(ctx context.Context, spec config.Spec, resolved config.Resolved, ports port.Ports, d deps) (Tools, error) {
	run := ports.Run
	if resolved.PythonPath == "" {
		return Tools{}, errors.New("no Python interpreter was selected")
	}
	var tools Tools
	for _, need := range spec.HostNeeds() {
		if err := checkNeed(ctx, spec, need, ports, d, &tools); err != nil {
			return Tools{}, err
		}
	}
	tools.Toolchain = releases.DetectToolchain(ctx, run)
	return tools, nil
}

func checkNeed(ctx context.Context, spec config.Spec, need catalog.Need, ports port.Ports, d deps, tools *Tools) error {
	run := ports.Run
	switch {
	case slices.Equal(need.Tools, []string{"git"}):
		if !d.host().CommandWorks(ctx, run, "git") {
			return errors.New("git is not on PATH; install Git, or leave out the git and pre-commit add-ons")
		}
	case slices.Equal(need.Tools, []string{"docker"}):
		return ports.Docker.Ready(ctx, spec.Name().String(), spec.Database())
	case slices.Equal(need.Tools, []string{"node"}):
		return requireNode(ctx, run, need.Minimum)
	case slices.Contains(need.Tools, "npm") || slices.Contains(need.Tools, "pnpm"):
		manager, ok := d.nodeManager(ctx, run)
		if !ok {
			return errors.New("the Tailwind stacks need Node.js with pnpm or npm on PATH to compile their CSS; " +
				"install Node.js, or choose a stack without Tailwind")
		}
		tools.NodeManager = manager
	default:
		for _, tool := range need.Tools {
			if d.host().CommandWorks(ctx, run, tool) {
				return nil
			}
		}
		return fmt.Errorf("%s is not on PATH", strings.Join(need.Tools, " or "))
	}
	return nil
}

func requireNode(ctx context.Context, run command.Runner, minimum int) error {
	output, err := run.Run(ctx, "", "node", "--version")
	if err != nil {
		return fmt.Errorf("the Tailwind stacks need Node.js %d or newer, and `node --version` failed: %w", minimum, err)
	}
	major, err := strconv.Atoi(strings.SplitN(strings.TrimPrefix(strings.TrimSpace(string(output)), "v"), ".", 2)[0])
	if err != nil {
		return fmt.Errorf("could not read the Node.js version from %q", strings.TrimSpace(string(output)))
	}
	if major < minimum {
		return fmt.Errorf("the Tailwind CSS %s build needs Node.js %d or newer; this machine has %s", tailwindVersion(), minimum, strings.TrimSpace(string(output)))
	}
	return nil
}

func (e *Env) projectPath() (string, error) {
	path := e.root
	if path == "" {
		path = e.cfg.Name().String()
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve the project directory %s: %w", path, err)
	}
	return absolute, nil
}

func (e *Env) prepareProjectDirectory() (string, error) {
	projectPath, err := e.projectPath()
	if err != nil {
		return "", err
	}

	fsys := e.machine().FS
	if err := checkDirectoryIsUsable(fsys, projectPath); err != nil {
		return "", err
	}

	_, statErr := fsys.Stat(projectPath)
	existed := statErr == nil

	if err := fsys.MkdirAll(projectPath); err != nil {
		return "", fmt.Errorf("create project directory: %w", err)
	}
	e.onRollback(func(context.Context) error { return removeProjectDirectory(fsys, projectPath, existed) })
	e.Step("Project directory created: " + projectPath)

	return projectPath, nil
}

func checkDirectoryIsUsable(fsys port.FS, path string) error {
	info, err := fsys.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect project directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("project path exists and is not a directory: %s", path)
	}

	entries, err := fsys.ReadDir(path)
	if err != nil {
		return fmt.Errorf("inspect project directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("project directory already exists and is not empty: %s", path)
	}
	return nil
}

func (p project) scriptFiles() fileSet {
	files := fileSet{}
	for _, goos := range []string{"linux", pyenv.Windows} {
		mode := execMode
		if pyenv.IsWindows(goos) {
			mode = fileMode
		}
		files.add(devStartScriptNameFor(goos), generateStartScript(goos, p.cfg.UsesTailwind()), mode)
	}
	return files
}

func (e *Env) writeDevStartScript(ctx context.Context) error {
	if err := e.emitGroup(groupScripts); err != nil {
		return err
	}
	for _, goos := range []string{"linux", pyenv.Windows} {
		e.Step("Created: " + devStartScriptNameFor(goos))
	}
	return nil
}

func removeProjectDirectory(fsys port.FS, projectPath string, existed bool) error {
	if !existed {
		return fsys.RemoveAll(projectPath)
	}
	entries, err := fsys.ReadDir(projectPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := fsys.RemoveAll(filepath.Join(projectPath, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

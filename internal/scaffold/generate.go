package scaffold

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	"github.com/vaibdix/django-stack/internal/plan"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/releases"
	"github.com/vaibdix/django-stack/internal/render"
)

const rollbackTimeout = 2 * time.Minute

func (e *Env) Generate(ctx context.Context) (err error) {
	if !e.checked {
		spec, err := probeAndParse(ctx, e.Cfg.Choices(), e.Resolved, e.commandRunner(), e.now())
		if err != nil {
			return err
		}
		tools, err := preflight(ctx, spec, e.Resolved, e.machine(), e.dependencies())
		if err != nil {
			return err
		}
		e.Toolchain, e.Node = tools.Toolchain, tools.NodeManager
	}
	if e.Resolved.Generated.IsZero() {
		e.Resolved.Generated = e.now()
	}

	if err := e.resolveSecrets(); err != nil {
		return err
	}

	if err := e.prepareProjectDirectory(); err != nil {
		return err
	}

	defer func() { err = e.afterRun(ctx, err) }()

	e.Step("Package manager: " + e.Toolchain.String())

	err = e.runPhase(ctx, false)
	var verification *VerificationError
	if err != nil && !errors.As(err, &verification) {
		return err
	}
	if err := e.publish(); err != nil {
		return err
	}
	if err := e.runPhase(ctx, true); err != nil {
		return err
	}
	if err != nil {
		return err
	}

	e.reportPerformanceTip()
	e.Step("Django project setup complete!")
	return nil
}

func (e *Env) runPhase(ctx context.Context, published bool) error {
	nodes, err := e.nodes(published)
	if err != nil {
		return err
	}
	label := labels()
	err = graph.Run(ctx, e, nodes, graph.Hooks{
		Started:  func(name string) { e.started(name, label[name]) },
		Finished: func(name string, took time.Duration, err error) { e.finished(name, label[name], took, err) },
		Clock:    e.now,
		Limit:    plan.Concurrency,
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
	return err
}

func (e *Env) afterRun(ctx context.Context, err error) error {
	var verification *VerificationError
	if err == nil || errors.As(err, &verification) {
		return err
	}
	return e.rollback(ctx, err)
}

func (e *Env) onRollback(undo func(context.Context) error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.undo = append(e.undo, undo)
}

func (e *Env) rollback(ctx context.Context, cause error) error {
	if e.keepOnFailure {
		if err := e.publish(); err != nil {
			return fmt.Errorf("%w\n\nThe partial project was kept at %s: %w", cause, e.dir, err)
		}
		return fmt.Errorf("%w\n\nThe partial project was kept at %s", cause, e.target)
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

func Preflight(ctx context.Context, spec config.Spec, resolved config.Resolved, ports port.Ports) (render.Tools, error) {
	return preflight(ctx, spec, resolved, adapter.Complete(ports), deps{}.withDefaults())
}

func preflight(ctx context.Context, spec config.Spec, resolved config.Resolved, ports port.Ports, d deps) (render.Tools, error) {
	run := ports.Run
	if resolved.PythonPath == "" {
		return render.Tools{}, errors.New("no Python interpreter was selected")
	}
	var tools render.Tools
	for _, need := range spec.HostNeeds() {
		if err := checkNeed(ctx, spec, need, ports, d, &tools); err != nil {
			return render.Tools{}, err
		}
	}
	tools.Toolchain = releases.DetectToolchain(ctx, run)
	return tools, nil
}

func checkNeed(ctx context.Context, spec config.Spec, need catalog.Need, ports port.Ports, d deps, tools *render.Tools) error {
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
		return fmt.Errorf("the Tailwind CSS %s build needs Node.js %d or newer; this machine has %s", render.TailwindVersion(), minimum, strings.TrimSpace(string(output)))
	}
	return nil
}

func (e *Env) projectPath() (string, error) {
	path := e.root
	if path == "" {
		path = e.Cfg.Name().String()
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve the project directory %s: %w", path, err)
	}
	return absolute, nil
}

func (e *Env) prepareProjectDirectory() error {
	target, err := e.projectPath()
	if err != nil {
		return err
	}

	fsys := e.machine().FS
	if err := checkDirectoryIsUsable(fsys, target); err != nil {
		return err
	}
	_, statErr := fsys.Stat(target)
	e.target, e.targetExisted = target, statErr == nil

	dir := target
	if e.Toolchain == pyenv.UV {
		dir, err = e.stagingPath(target)
		if err != nil {
			return err
		}
	}
	existed := dir == target && e.targetExisted
	if err := fsys.MkdirAll(dir); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}
	e.onRollback(func(context.Context) error { return removeProjectDirectory(fsys, dir, existed) })
	e.dir = dir
	if dir == target {
		e.Step("Project directory created: " + target)
	} else {
		e.Step("Building the project in " + filepath.Base(dir) + " next to " + target + "; it moves into place once it passes its checks")
	}
	return nil
}

func (e *Env) stagingPath(target string) (string, error) {
	suffix := make([]byte, stagingSuffixBytes)
	if _, err := io.ReadFull(e.randomness(), suffix); err != nil {
		return "", fmt.Errorf("name the staging folder: %w", err)
	}
	return filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".django-stack-"+hex.EncodeToString(suffix)), nil
}

const stagingSuffixBytes = 4

func (e *Env) staged() bool { return e.dir != e.target }

func (e *Env) publish() error {
	if !e.staged() {
		return nil
	}
	fsys := e.machine().FS
	if err := removeCompiledPython(fsys, e.dir); err != nil {
		return fmt.Errorf("clear compiled Python before publishing: %w", err)
	}
	if e.targetExisted {
		if err := checkDirectoryIsUsable(fsys, e.target); err != nil {
			return err
		}
		if err := fsys.RemoveAll(e.target); err != nil {
			return fmt.Errorf("replace the empty folder %s: %w", e.target, err)
		}
	}
	if err := fsys.Rename(e.dir, e.target); err != nil {
		return fmt.Errorf("move the project into %s: %w", e.target, err)
	}
	e.dir = e.target
	target, existed := e.target, e.targetExisted
	e.onRollback(func(context.Context) error { return removeProjectDirectory(fsys, target, existed) })
	e.Step("Moved the checked project into " + e.target)
	return nil
}

func removeCompiledPython(fsys port.FS, dir string) error {
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "node_modules" || entry.Name() == ".git" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if entry.Name() == "__pycache__" {
			if err := fsys.RemoveAll(path); err != nil {
				return err
			}
			continue
		}
		if err := removeCompiledPython(fsys, path); err != nil {
			return err
		}
	}
	return nil
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

func (e *Env) writeDevStartScript(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupScripts); err != nil {
		return err
	}
	for _, goos := range []string{"linux", pyenv.Windows} {
		e.Step("Created: " + render.DevStartScriptNameFor(goos))
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

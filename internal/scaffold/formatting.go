package scaffold

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

type formattingCommand struct {
	Stage  string
	Module string
	Args   []string
}

type scaffoldFormattingPlan struct {
	Packages      []string
	Commands      []formattingCommand
	TemplateFiles []string
}

// runPythonModuleCommand runs `python -m module args...` in the project.
var runPythonModuleCommand = func(ctx context.Context, e *Env, projectPath, python, module string, args ...string) ([]byte, error) {
	return e.runIn(ctx, projectPath, python, append([]string{"-m", module}, args...)...)
}

// installScratchFormatters installs packages into a venv of their own in a
// temporary directory, with the run's interpreter and toolchain, and returns
// that venv's interpreter and the function that deletes the directory.
var installScratchFormatters = func(ctx context.Context, e *Env, packages []string) (string, func() error, error) {
	dir, err := os.MkdirTemp("", "django-stack-formatters-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to make a directory for the formatters: %w", err)
	}
	remove := func() error { return os.RemoveAll(dir) }

	if e.toolchain == pyenv.UV {
		_, err = e.runIn(ctx, dir, "uv", "venv", "--python", e.Cfg.SelectedPythonPath, pyenv.VenvDir)
	} else {
		_, err = e.runIn(ctx, dir, e.Cfg.SelectedPythonPath, "-m", "venv", pyenv.VenvDir)
	}
	if err == nil {
		name, args := pyenv.PackageInstallArgs(e.toolchain, dir, packages...)
		_, err = e.runIn(ctx, dir, name, args...)
	}
	if err != nil {
		return "", nil, errors.Join(
			fmt.Errorf("failed to install %s for formatting: %w", strings.Join(packages, " and "), err), remove())
	}
	return pyenv.InterpreterPath(dir), remove, nil
}

const djlintReformattedExitCode = 1

func djlintDidReformat(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == djlintReformattedExitCode
}

func (e *Env) declaredFormattingPackages() []string {
	var declared []string
	if e.Cfg.SetupRuff {
		declared = append(declared, PkgRuff)
	}
	if e.Cfg.CreateTemplates {
		declared = append(declared, PkgDjLint)
	}
	return declared
}

func buildScaffoldFormattingPlan(projectPath string) (scaffoldFormattingPlan, error) {
	templateFiles, err := findTemplateFilesForFormatting(projectPath)
	if err != nil {
		return scaffoldFormattingPlan{}, err
	}

	packages := []string{PkgRuff}
	if len(templateFiles) > 0 {
		packages = append(packages, PkgDjLint)
	}

	return scaffoldFormattingPlan{
		Packages: packages,
		Commands: []formattingCommand{
			{
				Stage:  "ruff autofix",
				Module: "ruff",
				// --fix-only: this stage fixes and never lints. Leftover findings are
				// the verify step's job (with Ruff chosen), and unparseable code is
				// caught by ruff format below, so a non-zero exit here is ruff itself
				// failing.
				Args: []string{"check", "--fix-only", "--select", "I,F401,UP", "."},
			},
			{
				Stage:  "ruff format",
				Module: "ruff",
				Args:   []string{"format", "."},
			},
		},
		TemplateFiles: templateFiles,
	}, nil
}

func findTemplateFilesForFormatting(projectPath string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(projectPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case pyenv.VenvDir, ".git", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}
		templateSegment := string(filepath.Separator) + TemplatesDir + string(filepath.Separator)
		if strings.Contains(path, templateSegment) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

// formatterInterpreters maps each formatter to the interpreter that runs it.
// A formatter the project declares runs from the project's venv, at the
// version the project records. The others go into a scratch venv outside the
// project, so the project's venv holds only what it declares and a formatter
// it never asked for cannot sway the versions it resolves.
func (e *Env) formatterInterpreters(ctx context.Context, projectPath string, formatters []string) (map[string]string, func() error, error) {
	declared := e.declaredFormattingPackages()
	pythons := make(map[string]string, len(formatters))
	var scratch []string
	for _, formatter := range formatters {
		if slices.Contains(declared, formatter) {
			pythons[formatter] = pyenv.InterpreterPath(projectPath)
		} else {
			scratch = append(scratch, formatter)
		}
	}
	if len(scratch) == 0 {
		return pythons, func() error { return nil }, nil
	}

	e.Step(fmt.Sprintf("Formatting with %s from a temporary environment, outside the project.", strings.Join(scratch, " and ")))
	python, remove, err := installScratchFormatters(ctx, e, scratch)
	if err != nil {
		return nil, nil, err
	}
	for _, formatter := range scratch {
		pythons[formatter] = python
	}
	return pythons, remove, nil
}

func (e *Env) formatGeneratedCode(projectPath string) (err error) {
	if !e.Cfg.SetupRuff {
		if err := e.dropLintSuppressions(projectPath); err != nil {
			return fmt.Errorf("failed to drop lint suppressions: %w", err)
		}
	}

	plan, planErr := buildScaffoldFormattingPlan(projectPath)
	if planErr != nil {
		return fmt.Errorf("failed to discover generated templates for formatting: %w", planErr)
	}

	ctx, cancel := e.timeoutContext()
	defer cancel()

	pythons, remove, err := e.formatterInterpreters(ctx, projectPath, plan.Packages)
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := remove(); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to remove the formatters' temporary environment: %w", removeErr))
		}
	}()

	for _, command := range plan.Commands {
		output, runErr := runPythonModuleCommand(ctx, e, projectPath, pythons[command.Module], command.Module, command.Args...)
		if runErr == nil {
			continue
		}
		return fmt.Errorf("failed during %s: %w\nOutput: %s", command.Stage, runErr, string(output))
	}

	if len(plan.TemplateFiles) == 0 {
		e.Step("No HTML templates found for djlint reformat.")
		e.Step("Generated code and templates formatted.")
		return nil
	}

	templateArgs := append([]string{"--reformat", "--profile=django"}, plan.TemplateFiles...)
	output, runErr := runPythonModuleCommand(ctx, e, projectPath, pythons[PkgDjLint], PkgDjLint, templateArgs...)
	if runErr != nil && !djlintDidReformat(runErr) {
		return fmt.Errorf("failed during template formatting: %w\nOutput: %s", runErr, string(output))
	}

	e.Step("Generated code and templates formatted.")
	return nil
}

// lintSuppression matches a trailing "# noqa" comment. They exist for the
// rules this project's Ruff configuration enables (S105 for test passwords,
// F403/F405 for the settings star import); with Ruff not chosen no linter is
// configured, they suppress nothing, and Ruff's defaults would report each one
// as unused (RUF100) if it were added later.
var lintSuppression = regexp.MustCompile(`[ \t]+# noqa(: [A-Z]+[0-9]+(,[A-Z]+[0-9]+)*)?[ \t]*$`)

func (e *Env) dropLintSuppressions(projectPath string) error {
	root, err := os.OpenRoot(projectPath)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	return fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case pyenv.VenvDir, ".git", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".py" {
			return nil
		}
		data, err := root.ReadFile(path)
		if err != nil {
			return err
		}
		lines := strings.Split(string(data), "\n")
		changed := false
		for i, line := range lines {
			if stripped := lintSuppression.ReplaceAllString(line, ""); stripped != line {
				lines[i], changed = stripped, true
			}
		}
		if !changed {
			return nil
		}
		return root.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
	})
}

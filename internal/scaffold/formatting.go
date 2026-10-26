package scaffold

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
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

func runPythonModule(ctx context.Context, e *Env, projectPath, python, module string, args ...string) ([]byte, error) {
	return e.runIn(ctx, projectPath, python, append([]string{"-m", module}, args...)...)
}

func installScratchFormatters(ctx context.Context, e *Env, packages []string) (string, func() error, error) {
	dir, err := os.MkdirTemp("", "django-stack-formatters-")
	if err != nil {
		return "", nil, fmt.Errorf("make a directory for the formatters: %w", err)
	}
	remove := func() error { return os.RemoveAll(dir) }

	if e.toolchain == pyenv.UV {
		_, err = e.runIn(ctx, dir, "uv", "venv", "--python", e.resolved.PythonPath, pyenv.VenvDir)
	} else {
		_, err = e.runIn(ctx, dir, e.resolved.PythonPath, "-m", "venv", pyenv.VenvDir)
	}
	if err == nil {
		name, args := pyenv.PackageInstallArgs(e.toolchain, dir, packages...)
		_, err = e.runIn(ctx, dir, name, args...)
	}
	if err != nil {
		return "", nil, errors.Join(
			fmt.Errorf("install %s for formatting: %w", strings.Join(packages, " and "), err), remove())
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
	if e.cfg.Has(config.Ruff) {
		declared = append(declared, pkgRuff)
	}
	if e.cfg.CreateTemplates() {
		declared = append(declared, pkgDjLint)
	}
	return declared
}

func buildScaffoldFormattingPlan(projectPath string) (scaffoldFormattingPlan, error) {
	templateFiles, err := findTemplateFilesForFormatting(projectPath)
	if err != nil {
		return scaffoldFormattingPlan{}, err
	}

	packages := []string{pkgRuff}
	if len(templateFiles) > 0 {
		packages = append(packages, pkgDjLint)
	}

	return scaffoldFormattingPlan{
		Packages: packages,
		Commands: []formattingCommand{
			{
				Stage:  "ruff autofix",
				Module: "ruff",

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
		templateSegment := string(filepath.Separator) + templatesDir + string(filepath.Separator)
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
	python, remove, err := e.dependencies().scratchTools(ctx, e, scratch)
	if err != nil {
		return nil, nil, err
	}
	for _, formatter := range scratch {
		pythons[formatter] = python
	}
	return pythons, remove, nil
}

func (e *Env) formatGeneratedCode(ctx context.Context) (err error) {
	plan, planErr := buildScaffoldFormattingPlan(e.dir)
	if planErr != nil {
		return fmt.Errorf("discover generated templates for formatting: %w", planErr)
	}
	plan.Commands = append(e.typingImportCommands(), plan.Commands...)

	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	pythons, remove, err := e.formatterInterpreters(ctx, e.dir, plan.Packages)
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := remove(); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("remove the formatters' temporary environment: %w", removeErr))
		}
	}()

	for _, command := range plan.Commands {
		output, runErr := e.dependencies().runModule(ctx, e, e.dir, pythons[command.Module], command.Module, command.Args...)
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
	output, runErr := e.dependencies().runModule(ctx, e, e.dir, pythons[pkgDjLint], pkgDjLint, templateArgs...)
	if runErr != nil && !djlintDidReformat(runErr) {
		return fmt.Errorf("failed during template formatting: %w\nOutput: %s", runErr, string(output))
	}

	e.Step("Generated code and templates formatted.")
	return nil
}

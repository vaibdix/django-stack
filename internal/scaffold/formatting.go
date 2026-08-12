package scaffold

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

type formattingCommand struct {
	Stage  string
	Module string
	Args   []string

	Advisory bool
}

type scaffoldFormattingPlan struct {
	Packages      []string
	Commands      []formattingCommand
	TemplateFiles []string
}

var installFormattingPackages = func(e *Env, projectPath string, packages ...string) error {
	return e.installDevPackages(projectPath, "scaffold formatters (ruff + djlint)", packages...)
}

var runPythonModuleCommand = func(ctx context.Context, e *Env, projectPath, module string, args ...string) ([]byte, error) {
	pythonPath := pyenv.InterpreterPath(projectPath)
	return e.runIn(ctx, projectPath, pythonPath, append([]string{"-m", module}, args...)...)
}

const djlintReformattedExitCode = 1

func djlintDidReformat(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == djlintReformattedExitCode
}

func scaffoldFormattingPackages() []string {
	return []string{PkgRuff, PkgDjLint}
}

func buildScaffoldFormattingPlan(projectPath string) (scaffoldFormattingPlan, error) {
	templateFiles, err := findTemplateFilesForFormatting(projectPath)
	if err != nil {
		return scaffoldFormattingPlan{}, err
	}

	return scaffoldFormattingPlan{
		Packages: scaffoldFormattingPackages(),
		Commands: []formattingCommand{
			{
				Stage:    "ruff autofix",
				Module:   "ruff",
				Args:     []string{"check", "--fix", "--select", "I,F401,UP", "."},
				Advisory: true,
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

func (e *Env) formatGeneratedCode(projectPath string) error {
	plan, planErr := buildScaffoldFormattingPlan(projectPath)
	if planErr != nil {
		return fmt.Errorf("failed to discover generated templates for formatting: %w", planErr)
	}

	if err := installFormattingPackages(e, projectPath, plan.Packages...); err != nil {
		return err
	}

	ctx, cancel := e.timeoutContext()
	defer cancel()

	for _, command := range plan.Commands {
		output, runErr := runPythonModuleCommand(ctx, e, projectPath, command.Module, command.Args...)
		if runErr == nil {
			continue
		}
		if command.Advisory {
			e.Step(fmt.Sprintf("Warning: %s reported issues it could not fix: %v\nOutput: %s",
				command.Stage, runErr, string(output)))
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
	output, err := runPythonModuleCommand(ctx, e, projectPath, "djlint", templateArgs...)
	if err != nil && !djlintDidReformat(err) {
		return fmt.Errorf("failed during template formatting: %w\nOutput: %s", err, string(output))
	}

	e.Step("Generated code and templates formatted.")
	return nil
}

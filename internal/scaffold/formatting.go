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

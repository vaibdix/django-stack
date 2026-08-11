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
}

type scaffoldFormattingPlan struct {
	Packages      []string
	Commands      []formattingCommand
	TemplateFiles []string
}

// installFormattingPackages is a variable so tests can substitute the install
// step instead of reaching PyPI.
var installFormattingPackages = func(e *Env, projectPath string, packages ...string) error {
	return e.installPackages(projectPath, "scaffold formatters (ruff + djlint)", packages...)
}

// runPythonModuleCommand runs `python -m <module>` in the project's virtualenv.
//
// It goes through the Env's runner, like every other subprocess the scaffolder
// starts, so the context reaches the child. This call used to be a bare
// exec.Command with no context at all — and because the formatting pass is a
// mandatory step that runs ruff twice and djlint over every generated template,
// it was the longest-running uninterruptible work in the program: quitting the
// TUI mid-format left ruff rewriting files in a project the user had abandoned.
//
// It stays a package-level var so tests can substitute it without a virtualenv;
// see withPythonModuleRunner in export_test.go.
var runPythonModuleCommand = func(ctx context.Context, e *Env, projectPath, module string, args ...string) ([]byte, error) {
	pythonPath := pyenv.InterpreterPath(projectPath)
	return e.runIn(ctx, projectPath, pythonPath, append([]string{"-m", module}, args...)...)
}

// djlintReformattedExitCode is what `djlint --reformat` returns when it has
// rewritten at least one file.
//
// Rewriting files is the entire purpose of the call, so this is success. The
// scaffolder previously treated any non-zero exit as fatal, which meant every
// project generated with templates aborted at the final formatting step —
// reporting failure immediately after the formatter had done its job correctly.
//
// The codes are distinct enough to tell apart: 0 means nothing needed changing,
// 1 means files were reformatted, and anything else (2 for a missing file or a
// usage error) is a genuine failure that must still abort.
const djlintReformattedExitCode = 1

// djlintDidReformat reports whether err is djlint signalling that it reformatted
// files rather than that it failed.
//
// errors.As, not a type assertion: the runner wraps the exit error in a
// command.Error to carry the captured output, so the *exec.ExitError is one
// Unwrap down rather than at the top. TestDjlintReformatIsRecognisedThroughTheRunner
// pins that, because a type assertion here would compile, always be false, and
// abort every scaffold that generated a template.
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
				Stage:  "ruff autofix",
				Module: "ruff",
				Args:   []string{"check", "--fix", "."},
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
		if runErr != nil {
			return fmt.Errorf("failed during %s: %w\nOutput: %s", command.Stage, runErr, string(output))
		}
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

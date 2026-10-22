package scaffold

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/report"
)

func TestBuildScaffoldFormattingPlanIncludesMandatoryPackagesAndTemplateFiles(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectPath, "templates"), 0755); err != nil {
		t.Fatalf("failed to create global templates dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectPath, "library", "templates", "library"), 0755); err != nil {
		t.Fatalf("failed to create app templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "templates", "base.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("failed to write global template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "library", "templates", "library", "book_list.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("failed to write app template: %v", err)
	}

	plan, err := buildScaffoldFormattingPlan(projectPath)
	if err != nil {
		t.Fatalf("buildScaffoldFormattingPlan failed: %v", err)
	}
	packages, commands, templateFiles := plan.Packages, plan.Commands, plan.TemplateFiles

	if len(packages) != 2 || packages[0] != "ruff" || packages[1] != "djlint" {
		t.Fatalf("expected mandatory formatter packages [ruff djlint], got %#v", packages)
	}
	if len(commands) != 2 {
		t.Fatalf("expected two mandatory Python formatting commands, got %#v", commands)
	}
	if commands[0].Module != "ruff" || !strings.HasPrefix(strings.Join(commands[0].Args, " "), "check --fix-only --select ") {
		t.Fatalf("unexpected first formatting command: %#v", commands[0])
	}
	if commands[0].Args[1] != "--fix-only" {
		t.Fatalf("the autofix pass must use --fix-only: a leftover lint finding aborting a scaffold is the bug "+
			"that broke every project which did not enable the Ruff feature, got %#v", commands[0].Args)
	}
	if commands[1].Module != "ruff" || strings.Join(commands[1].Args, " ") != "format ." {
		t.Fatalf("unexpected second formatting command: %#v", commands[1])
	}
	if len(templateFiles) != 2 {
		t.Fatalf("expected two template files for djlint, got %#v", templateFiles)
	}
}

func TestBuildScaffoldFormattingPlanSkipsTemplatesWhenNoneExist(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()

	plan, err := buildScaffoldFormattingPlan(projectPath)
	if err != nil {
		t.Fatalf("buildScaffoldFormattingPlan failed: %v", err)
	}
	templateFiles := plan.TemplateFiles
	if len(templateFiles) != 0 {
		t.Fatalf("expected no template files when project has none, got %#v", templateFiles)
	}
	if !slices.Equal(plan.Packages, []string{"ruff"}) {
		t.Errorf("with no templates there is nothing for djlint to format, yet the plan installs %v", plan.Packages)
	}
}

func TestScaffoldFormattingRunsUndeclaredFormattersOutsideTheProjectVenv(t *testing.T) {
	const scratchPython = "/scratch/.venv/bin/python"

	cases := []struct {
		name        string
		cfg         config.Project
		wantScratch []string
		wantPython  map[string]string
	}{
		{
			name:        "neither declared",
			cfg:         config.Project{},
			wantScratch: []string{"ruff", "djlint"},
			wantPython:  map[string]string{"ruff": "scratch", "djlint": "scratch"},
		},
		{
			name:        "ruff declared, templates not",
			cfg:         config.Project{SetupRuff: true},
			wantScratch: []string{"djlint"},
			wantPython:  map[string]string{"ruff": "project", "djlint": "scratch"},
		},
		{
			name:       "both declared",
			cfg:        config.Project{SetupRuff: true, CreateTemplates: true},
			wantPython: map[string]string{"ruff": "project", "djlint": "project"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projectPath := t.TempDir()
			writeTemplate(t, projectPath)
			projectPython := pyenv.InterpreterPath(projectPath)

			var scratch []string
			removed := 0
			defer withScratchFormatters(func(_ context.Context, _ *Env, packages []string) (string, func() error, error) {
				scratch = packages
				return scratchPython, func() error { removed++; return nil }, nil
			})()

			var called []string
			var sawContext bool
			defer withPythonModuleRunner(func(ctx context.Context, _ *Env, dir, python, module string, args ...string) ([]byte, error) {
				sawContext = ctx != nil && ctx.Err() == nil
				if dir != projectPath {
					t.Errorf("%s ran in %s, not the project", module, dir)
				}
				where := map[string]string{projectPython: "project", scratchPython: "scratch"}[python]
				if where != tc.wantPython[module] {
					t.Errorf("%s ran from %q, want the %s interpreter", module, python, tc.wantPython[module])
				}
				called = append(called, module+" "+strings.Join(args, " "))
				return []byte("ok"), nil
			})()

			env := &Env{cfg: tc.cfg, reporter: report.Nop{}}
			if err := env.formatGeneratedCode(projectPath); err != nil {
				t.Fatalf("formatGeneratedCode failed: %v", err)
			}

			if !slices.Equal(scratch, tc.wantScratch) {
				t.Errorf("scratch environment got %v, want %v", scratch, tc.wantScratch)
			}
			if wantRemoved := min(len(tc.wantScratch), 1); removed != wantRemoved {
				t.Errorf("the scratch environment was removed %d times, want %d", removed, wantRemoved)
			}
			if len(called) != 3 || !strings.HasPrefix(called[2], "djlint --reformat --profile=django") {
				t.Fatalf("expected ruff check, ruff format, and djlint --reformat, got %#v", called)
			}
			if !sawContext {
				t.Error("the formatting commands ran without a live context; quitting mid-format would not stop ruff or djlint")
			}
		})
	}
}

func TestScaffoldFormattingRemovesTheScratchVenvWhenAFormatterFails(t *testing.T) {
	projectPath := t.TempDir()
	removed := false
	defer withScratchFormatters(func(context.Context, *Env, []string) (string, func() error, error) {
		return "/scratch/python", func() error { removed = true; return errors.New("directory busy") }, nil
	})()
	defer withPythonModuleRunner(func(context.Context, *Env, string, string, string, ...string) ([]byte, error) {
		return []byte("boom"), errors.New("exit status 2")
	})()

	err := (&Env{reporter: report.Nop{}}).formatGeneratedCode(projectPath)
	if !removed {
		t.Error("the scratch environment outlived a failed format")
	}
	if err == nil || !strings.Contains(err.Error(), "ruff autofix") || !strings.Contains(err.Error(), "directory busy") {
		t.Errorf("formatGeneratedCode = %v, want both the ruff failure and the removal failure", err)
	}
}

func writeTemplate(t *testing.T, projectPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(projectPath, "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "templates", "base.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestRunMandatoryScaffoldFormattingFailsWhenRuffItselfFails(t *testing.T) {
	projectPath := t.TempDir()
	env := &Env{cfg: config.Project{SetupRuff: true, CreateTemplates: true}, reporter: report.Nop{}}

	restoreRunner := withPythonModuleRunner(func(_ context.Context, _ *Env, _, _, module string, args ...string) ([]byte, error) {
		if module == "ruff" && len(args) >= 2 && args[0] == "check" {
			return []byte("ruff failed: invalid configuration"), errors.New("exit status 2")
		}
		return []byte("ok"), nil
	})
	defer restoreRunner()

	err := env.formatGeneratedCode(projectPath)
	if err == nil || !strings.Contains(err.Error(), "ruff autofix") || !strings.Contains(err.Error(), "invalid configuration") {
		t.Fatalf("formatGeneratedCode = %v, want the ruff failure and its output", err)
	}
}

func TestRunMandatoryScaffoldFormattingFailsOnTemplateFormattingError(t *testing.T) {
	projectPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectPath, "templates"), 0755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "templates", "base.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	env := &Env{cfg: config.Project{SetupRuff: true, CreateTemplates: true}}

	restoreRunner := withPythonModuleRunner(func(_ context.Context, _ *Env, _, _, module string, _ ...string) ([]byte, error) {
		if module == "djlint" {
			return []byte("template parse error"), errors.New("exit status 1")
		}
		return []byte("ok"), nil
	})
	defer restoreRunner()

	err := env.formatGeneratedCode(projectPath)
	if err == nil {
		t.Fatalf("expected formatting failure when djlint fails")
	}
	if !strings.Contains(err.Error(), "template formatting") {
		t.Fatalf("expected error to mention template formatting stage, got %v", err)
	}
}

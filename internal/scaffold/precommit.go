package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func (e *Env) applyPreCommit(projectPath string) error {

	config := e.generatePreCommitConfig()

	configPath := filepath.Join(projectPath, PreCommitConfigFile)
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", PreCommitConfigFile, err)
	}
	e.Step("Created: " + PreCommitConfigFile)

	pythonPath := pyenv.InterpreterPath(projectPath)
	ctx, cancel := e.timeoutContext()
	defer cancel()

	if _, err := e.runIn(ctx, projectPath, pythonPath, "-m", "pre_commit", "install"); err != nil {
		e.Step("Warning: pre-commit install failed: " + err.Error())
		return nil
	}

	e.Step("Pre-commit hooks installed")
	return nil
}

const defaultDjangoTrack = "6.1"

func (e *Env) djangoTrack() string {
	if e.Cfg.DjangoVersion == "" {
		return defaultDjangoTrack
	}
	parts := strings.SplitN(e.Cfg.DjangoVersion, ".", 3)
	if len(parts) < 2 {
		return defaultDjangoTrack
	}
	return parts[0] + "." + parts[1]
}

const (
	PreCommitHooksRev = "v6.0.0"
	DjangoUpgradeRev  = "1.33.0"
)

type preCommitData struct {
	RuffEnabled       bool
	MypyEnabled       bool
	UVAvailable       bool
	Templates         bool
	DjangoTrack       string
	PreCommitHooksRev string
	DjangoUpgradeRev  string
	MypyEntry         string
	RuffCheckEntry    string
	RuffFormatEntry   string
	DjLintEntry       string
}

func (e *Env) generatePreCommitConfig() string {
	return renderPreCommitConfig(preCommitData{
		RuffEnabled: e.Cfg.SetupRuff,
		MypyEnabled: e.Cfg.SetupMypy,
		UVAvailable: e.toolchain == pyenv.UV,
		Templates:   e.Cfg.CreateTemplates,
		DjangoTrack: e.djangoTrack(),
	}, runtime.GOOS == "windows")
}

func renderPreCommitConfig(data preCommitData, windows bool) string {
	data.PreCommitHooksRev = PreCommitHooksRev
	data.DjangoUpgradeRev = DjangoUpgradeRev
	data.MypyEntry = venvToolEntry(data.UVAvailable, windows, "mypy", ".")
	data.RuffCheckEntry = venvToolEntry(data.UVAvailable, windows, "ruff", "check", "--fix")
	data.RuffFormatEntry = venvToolEntry(data.UVAvailable, windows, "ruff", "format")
	data.DjLintEntry = venvToolEntry(data.UVAvailable, windows, "djlint", "--lint", "--profile=django")
	return render("pre-commit-config.yaml.tmpl", data)
}

func venvToolEntry(uv, windows bool, module string, args ...string) string {
	command := []string{"uv", "run", module}
	if !uv {
		python := ".venv/bin/python"
		if windows {
			python = ".venv/Scripts/python.exe"
		}
		command = []string{python, "-m", module}
	}
	return strings.Join(append(command, args...), " ")
}

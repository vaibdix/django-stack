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
	if err := e.installDevPackages(projectPath, "pre-commit", PkgPreCommit); err != nil {
		return err
	}

	config := e.generatePreCommitConfig()

	configPath := filepath.Join(projectPath, PreCommitConfigFile)
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", PreCommitConfigFile, err)
	}
	e.Step("Created: " + PreCommitConfigFile)

	pythonPath := pyenv.InterpreterPath(projectPath)
	ctx, cancel := e.timeoutContext()
	defer cancel()

	if output, err := e.runIn(ctx, projectPath, pythonPath, "-m", "pre_commit", "autoupdate"); err != nil {
		e.Step(fmt.Sprintf("Warning: pre-commit autoupdate failed, so the pinned hook revisions were kept: %v\nOutput: %s", err, string(output)))
	} else {
		e.Step("Pre-commit hook revisions updated to the latest releases")
	}

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

type preCommitData struct {
	RuffEnabled   bool
	MypyEnabled   bool
	UVAvailable   bool
	DjangoTrack   string
	DjLintVersion string
	MypyEntry     string
}

func (e *Env) generatePreCommitConfig() string {
	return renderPreCommitConfig(preCommitData{
		RuffEnabled:   e.Cfg.SetupRuff,
		MypyEnabled:   e.Cfg.SetupMypy,
		UVAvailable:   pyenv.UVAvailable(),
		DjangoTrack:   e.djangoTrack(),
		DjLintVersion: DjLintVersion,
	}, runtime.GOOS == "windows")
}

func renderPreCommitConfig(data preCommitData, windows bool) string {
	data.MypyEntry = venvToolEntry(data.UVAvailable, windows, "mypy", ".")
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

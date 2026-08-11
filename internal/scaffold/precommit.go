package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func (e *Env) applyPreCommit(projectPath string) error {
	if err := e.installPackages(projectPath, "pre-commit", PkgPreCommit); err != nil {
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

	cmd := exec.CommandContext(ctx, pythonPath, "-m", "pre_commit", "install")
	cmd.Dir = projectPath
	if runtime.GOOS == "windows" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			e.Step("Warning: pre-commit install failed: " + err.Error())
		}
	} else {
		if output, err := cmd.CombinedOutput(); err != nil {
			e.Step(fmt.Sprintf("Warning: pre-commit install failed: %v\nOutput: %s", err, string(output)))
		}
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

func (e *Env) generatePreCommitConfig() string {
	return render("pre-commit-config.yaml.tmpl", struct {
		RuffEnabled bool
		DjangoTrack string
	}{
		RuffEnabled: e.Cfg.SetupRuff,
		DjangoTrack: e.djangoTrack(),
	})
}

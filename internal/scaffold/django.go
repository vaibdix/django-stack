package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func (e *Env) createDjangoProject(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	pythonVenvPath := pyenv.InterpreterPath(projectPath)
	if _, err := e.runIn(ctx, projectPath, pythonVenvPath, "-m", "django", "startproject", e.Cfg.ModuleName(), "."); err != nil {
		return fmt.Errorf("failed to create Django project: %w", err)
	}

	if err := annotateManageMain(projectPath); err != nil {
		return err
	}

	if err := e.stripCommentsFromDjangoBoilerplate(projectPath); err != nil {
		return err
	}

	e.Step(fmt.Sprintf("Django project '%s' created.", e.Cfg.ProjectName))
	return nil
}

func annotateManageMain(projectPath string) error {
	path := filepath.Join(projectPath, ManageFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", ManageFile, err)
	}

	content := strings.Replace(string(data), "def main():", "def main() -> None:", 1)
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write %s: %w", ManageFile, err)
	}
	return nil
}

func (e *Env) configureSettings(settingsPath string) error {
	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	envCfg := e.envConfig()
	se.InjectEnvConfig(envCfg)
	if err := se.InjectSecuritySettings(config.DjangoSupportsNativeCSP(e.Cfg.DjangoVersion), envCfg); err != nil {
		return err
	}

	if err := se.AddToListAfter(MiddlewareList, MiddlewareSecurity, MiddlewareWhiteNoise); err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", MiddlewareWhiteNoise, MiddlewareList, err)
	}

	if !e.Cfg.SetupMultiEnv && !se.Contains(MiddlewareBrowserReload) {
		se.AppendBlock(browserReloadBlock)
	}

	if err := se.AddProjectContextProcessor(e.Cfg.ModuleName()); err != nil {
		return err
	}

	if err := se.Save(); err != nil {
		return err
	}
	e.Step("Django settings configured.")
	if e.Cfg.SetupPydanticConf {
		e.Step("Environment variables configured via pydantic-settings.")
	} else {
		e.Step("Environment variables configured via django-environ.")
	}
	return nil
}

func (e *Env) reportPerformanceTip() {
	if !pyenv.UVAvailable() && runtime.GOOS == "windows" {
		e.Step(
			"Tip: Install 'uv' for faster package management: pip install uv")
	}
}

const browserReloadBlock = `
if DEBUG:
    INSTALLED_APPS = [*INSTALLED_APPS, '` + AppBrowserReload + `']
    MIDDLEWARE = [*MIDDLEWARE, '` + MiddlewareBrowserReload + `']
`

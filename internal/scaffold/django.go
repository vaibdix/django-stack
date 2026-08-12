package scaffold

import (
	"context"
	"fmt"
	"runtime"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func (e *Env) createDjangoProject(projectPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	pythonVenvPath := pyenv.InterpreterPath(projectPath)
	if _, err := e.runIn(ctx, projectPath, pythonVenvPath, "-m", "django", "startproject", e.Cfg.ModuleName(), "."); err != nil {
		return fmt.Errorf("failed to create Django project: %w", err)
	}

	e.Step(fmt.Sprintf("Django project '%s' created.", e.Cfg.ProjectName))
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

	if !se.Contains("'" + AppBrowserReload + "',") {
		if err := se.AddToList(InstalledApps, AppBrowserReload); err != nil {
			return fmt.Errorf("failed to add %s to %s: %w", AppBrowserReload, InstalledApps, err)
		}
	}
	if !se.Contains(MiddlewareBrowserReload) {
		if err := se.AddToList(MiddlewareList, MiddlewareBrowserReload); err != nil {
			return fmt.Errorf("failed to add %s to %s: %w", MiddlewareBrowserReload, MiddlewareList, err)
		}
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

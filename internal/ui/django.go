package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

func (m *Model) CreateDjangoProject(projectPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	pythonVenvPath := getPythonPath(projectPath)
	cmd := exec.CommandContext(ctx, pythonVenvPath, "-m", "django", "startproject", m.ProjectModuleName(), ".")
	cmd.Dir = projectPath
	m.UpdateProgress("Creating Django project...")

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create Django project: %w\nOutput: %s", err, string(output))
	}

	m.AppendStepMessage(fmt.Sprintf("Django project '%s' created.", m.ProjectName))
	return nil
}

func (m *Model) ConfigureDjangoSettings(settingsPath string) error {
	m.UpdateProgress("Configuring Django settings...")

	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	se.InjectEnvConfig(m.DatabaseType)

	if !se.Contains("PASSWORD_HASHERS") {
		se.InjectSecuritySettings()
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

	se.AddProjectContextProcessor(m.ProjectModuleName())

	if err := se.Save(); err != nil {
		return err
	}
	m.AppendStepMessage("Django settings configured.")
	m.AppendStepMessage("Environment variables configured via django-environ.")
	return nil
}

func (m *Model) ShowPerformanceTip() {
	if !isUvAvailable() && runtime.GOOS == "windows" {
		m.AppendStepMessage(
			"Tip: Install 'uv' for faster package management: pip install uv")
	}
}

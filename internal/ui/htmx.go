package ui

import (
	"fmt"
	"path/filepath"
)

func (m *Model) SetupHTMXIntegration(projectPath string) error {
	if !m.SetupHTMX {
		return nil
	}
	m.UpdateProgress("Setting up HTMX...")

	if err := m.installAndReport(projectPath, "django-htmx", PkgHTMX); err != nil {
		return err
	}

	settingsPath := filepath.Join(projectPath, m.ProjectModuleName(), SettingsFile)
	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	if !se.Contains("'" + AppHTMX + "',") {
		if err := se.AddToList(InstalledApps, AppHTMX); err != nil {
			return fmt.Errorf("failed to add %s to %s: %w", AppHTMX, InstalledApps, err)
		}
	}

	if !se.Contains(MiddlewareHTMX) {
		if err := se.AddToList(MiddlewareList, MiddlewareHTMX); err != nil {
			return fmt.Errorf("failed to add %s to %s: %w", MiddlewareHTMX, MiddlewareList, err)
		}
	}

	if err := se.Save(); err != nil {
		return err
	}
	m.AppendStepMessage(fmt.Sprintf("Added %s to %s and %s to %s.", AppHTMX, InstalledApps, "HtmxMiddleware", MiddlewareList))

	return nil
}

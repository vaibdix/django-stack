package scaffold

import (
	"fmt"
	"path/filepath"
)

func (e *Env) applyHTMX(projectPath string) error {
	settingsPath := filepath.Join(projectPath, e.Cfg.ModuleName(), SettingsFile)
	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	if err := se.AddToList(InstalledApps, AppHTMX); err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", AppHTMX, InstalledApps, err)
	}

	if err := se.AddToList(MiddlewareList, MiddlewareHTMX); err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", MiddlewareHTMX, MiddlewareList, err)
	}

	if err := se.Save(); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Added %s to %s and %s to %s.", AppHTMX, InstalledApps, "HtmxMiddleware", MiddlewareList))

	return nil
}

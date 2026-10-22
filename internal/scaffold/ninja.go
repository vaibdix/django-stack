package scaffold

import (
	"fmt"
	"path/filepath"
)

func (e *Env) applyNinja(projectPath string) error {
	settingsPath := filepath.Join(projectPath, e.Cfg.ModuleName(), SettingsFile)
	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	if err := se.AddToList(InstalledApps, AppNinja); err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", AppNinja, InstalledApps, err)
	}

	if err := se.Save(); err != nil {
		return err
	}

	if e.Cfg.AppName != "" {
		if err := e.writeAPIPackage(projectPath, generateNinjaPackage(e.Cfg.AppName, e.Cfg.ProjectName)); err != nil {
			return err
		}
	}

	return nil
}

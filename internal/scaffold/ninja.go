package scaffold

import (
	"fmt"
	"path/filepath"
)

func (e *Env) applyNinja(projectPath string) error {
	settingsPath := filepath.Join(projectPath, e.cfg.ModuleName(), settingsFile)
	se, err := newSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	if err := se.AddToList(installedApps, appNinja); err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", appNinja, installedApps, err)
	}

	if err := se.Save(); err != nil {
		return err
	}

	if e.cfg.AppName != "" {
		if err := e.writeAPIPackage(projectPath, generateNinjaPackage(e.cfg.AppName, e.cfg.ProjectName)); err != nil {
			return err
		}
	}

	return nil
}

package scaffold

import (
	"context"
	"fmt"
	"path/filepath"
)

func (e *Env) applyNinja(ctx context.Context) error {
	settingsPath := filepath.Join(e.dir, e.cfg.ModuleName(), settingsFile)
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

	if e.cfg.App != "" {
		if err := e.writeAPIPackage(e.dir, generateNinjaPackage(e.cfg.App, e.cfg.Name)); err != nil {
			return err
		}
	}

	return nil
}

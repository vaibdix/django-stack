package scaffold

import (
	"context"
	"fmt"
)

func (e *Env) applyNinja(ctx context.Context) error {
	se, err := e.settings()
	if err != nil {
		return err
	}

	if err := se.AddToList(installedApps, appNinja); err != nil {
		return fmt.Errorf("add %s to %s: %w", appNinja, installedApps, err)
	}

	if err := se.Save(); err != nil {
		return err
	}

	if e.cfg.App != "" {
		if err := e.writeAPIPackage(generateNinjaPackage(e.cfg.App, e.cfg.Name)); err != nil {
			return err
		}
	}

	return nil
}

package scaffold

import (
	"context"
	"fmt"
)

func (e *Env) applyHTMX(ctx context.Context) error {
	se, err := e.settings()
	if err != nil {
		return err
	}

	if err := se.AddToList(installedApps, appHTMX); err != nil {
		return fmt.Errorf("add %s to %s: %w", appHTMX, installedApps, err)
	}

	if err := se.AddToList(middlewareList, middlewareHTMX); err != nil {
		return fmt.Errorf("add %s to %s: %w", middlewareHTMX, middlewareList, err)
	}

	if err := se.Save(); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Added %s to %s and %s to %s.", appHTMX, installedApps, "HtmxMiddleware", middlewareList))

	return nil
}

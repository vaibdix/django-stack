package scaffold

import (
	"context"
)

func (e *Env) applyNinja(ctx context.Context) error {
	if e.Cfg.App().Ok() {
		if err := e.writeAPIPackage(ctx); err != nil {
			return err
		}
	}

	return nil
}

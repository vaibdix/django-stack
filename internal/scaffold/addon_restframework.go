package scaffold

import (
	"context"
)

func (e *Env) applyRESTFramework(ctx context.Context) error {
	if e.Cfg.App().Ok() {
		if err := e.writeAPIPackage(); err != nil {
			return err
		}
	}

	return nil
}

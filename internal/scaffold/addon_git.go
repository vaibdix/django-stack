package scaffold

import (
	"context"
	"fmt"
)

func (e *Env) initGitRepository(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if _, err := e.runIn(ctx, e.dir, "git", "init"); err != nil {
		return fmt.Errorf("initialize Git repository: %w", err)
	}
	e.Step("Git repository initialized.")
	return nil
}

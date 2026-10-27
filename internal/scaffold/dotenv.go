package scaffold

import (
	"context"
	"io"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) randomness() io.Reader { return e.machine().Rand }

func (e *Env) resolveSecrets() error {
	if e.Resolved.SecretKey == "" {
		key, err := render.GenerateSecretKey(e.randomness())
		if err != nil {
			return err
		}
		e.Resolved.SecretKey = key
	}
	if e.Cfg.Database().UsesDocker() && e.Resolved.Database == (config.DatabaseCredentials{}) {
		return e.prepareDatabaseCredentials()
	}
	return nil
}

func (e *Env) writeEnvFile(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupEnv); err != nil {
		return err
	}

	e.Step("Created: .env (with SECRET_KEY and database config)")
	return nil
}

func (e *Env) writeEnvExampleFile(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupEnvExample); err != nil {
		return err
	}

	e.Step("Created: .env.example")
	return nil
}

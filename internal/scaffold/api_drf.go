package scaffold

import (
	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) writeAPIPackage() error {
	if err := e.emitGroup(render.GroupAPI); err != nil {
		return err
	}
	e.Step("Created api/ package.")
	return nil
}

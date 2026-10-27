package scaffold

import (
	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) writeVSCodeConfig() error {
	return e.emitGroup(render.GroupWorkspace)
}

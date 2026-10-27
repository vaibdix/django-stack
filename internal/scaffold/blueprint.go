package scaffold

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/vaibdix/django-stack/internal/plan"
	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) emitGroup(ctx context.Context, g render.Group) error {
	if effect, ok := runningEffect(ctx); ok && !slices.Contains(effect.Produces, plan.Group(g)) {
		return fmt.Errorf("%s writes the %s files but does not declare them", effect.ID, g)
	}
	files, err := e.Files(g)
	if err != nil {
		return err
	}
	return e.emit(files)
}

func (e *Env) emit(files render.FileSet) error {
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		if err := e.write(filepath.FromSlash(rel), files[rel].Content, files[rel].Mode); err != nil {
			return err
		}
	}
	return nil
}

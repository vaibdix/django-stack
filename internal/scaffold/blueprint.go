package scaffold

import (
	"maps"
	"path/filepath"
	"slices"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) emitGroup(g render.Group) error {
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

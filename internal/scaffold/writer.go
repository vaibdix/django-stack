package scaffold

import (
	"io/fs"
	"maps"
	"path/filepath"
	"slices"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) write(rel string, content []byte, mode fs.FileMode) error {
	if err := e.machine().FS.WriteIn(e.dir, rel, content, mode); err != nil {
		return err
	}
	e.mu.Lock()
	e.written = append(e.written, filepath.ToSlash(rel))
	e.mu.Unlock()
	return nil
}

func (e *Env) writeFiles(dir string, files map[string]string) error {
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := e.write(filepath.Join(dir, name), []byte(files[name]), render.FileMode); err != nil {
			return err
		}
	}
	return nil
}

func (e *Env) read(rel string) ([]byte, error) {
	return e.machine().FS.ReadFile(filepath.Join(e.dir, rel))
}

package scaffold

import (
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
)

const (
	fileMode   fs.FileMode = 0o644
	secretMode fs.FileMode = 0o600
	execMode   fs.FileMode = 0o755
	dirMode    fs.FileMode = 0o755
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
		if err := e.write(filepath.Join(dir, name), []byte(files[name]), fileMode); err != nil {
			return err
		}
	}
	return nil
}

func (e *Env) read(rel string) ([]byte, error) {
	return e.machine().FS.ReadFile(filepath.Join(e.dir, rel))
}

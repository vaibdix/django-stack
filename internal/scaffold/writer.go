package scaffold

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
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
	root, err := os.OpenRoot(e.dir)
	if err != nil {
		return fmt.Errorf("open the project directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	if dir := filepath.Dir(rel); dir != "." {
		if err := root.MkdirAll(dir, dirMode); err != nil {
			return fmt.Errorf("create %s: %w", filepath.ToSlash(dir), err)
		}
	}
	if err := root.WriteFile(rel, content, mode); err != nil {
		return fmt.Errorf("write %s: %w", filepath.ToSlash(rel), err)
	}
	e.written = append(e.written, filepath.ToSlash(rel))
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
	return os.ReadFile(filepath.Join(e.dir, rel))
}

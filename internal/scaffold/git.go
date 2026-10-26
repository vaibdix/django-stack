package scaffold

import (
	"context"
)

func (p project) gitignoreFiles() fileSet {
	files := fileSet{}
	files.add(gitignoreFile, renderStatic("gitignore.tmpl"), fileMode)
	return files
}

func (e *Env) writeGitignore(ctx context.Context) error {
	if err := e.emit(e.gitignoreFiles()); err != nil {
		return err
	}
	e.Step("Created: " + gitignoreFile)
	return nil
}

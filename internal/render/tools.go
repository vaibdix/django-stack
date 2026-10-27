package render

import (
	"github.com/vaibdix/django-stack/internal/pyenv"
)

type Tools struct {
	Toolchain   pyenv.Toolchain
	NodeManager NodeManager
}

func (p Project) scriptFiles() FileSet {
	files := FileSet{}
	for _, goos := range []string{"linux", pyenv.Windows} {
		mode := execMode
		if pyenv.IsWindows(goos) {
			mode = FileMode
		}
		files.add(DevStartScriptNameFor(goos), generateStartScript(goos, p.Cfg.UsesTailwind()), mode)
	}
	return files
}

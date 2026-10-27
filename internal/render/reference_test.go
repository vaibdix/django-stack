package render

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/golden"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

var referenceDir = filepath.Join("..", "..", "reference")

func referenceChoices() config.Choices {
	return config.Choices{
		Name:     "my-shop",
		App:      "catalog",
		Django:   "6.1.2",
		Database: config.SQLite,
		Stack:    config.StackTailwindHTMX,
		AddOns:   everyAddOnBut(config.Ninja),
	}
}

func TestGoldenReferenceProject(t *testing.T) {
	t.Parallel()

	resolved := config.Resolved{
		PythonVersion: "Python 3.13.2",
		SecretKey:     "reference-secret-key-not-for-use",
		Generated:     time.Date(2026, time.October, 27, 12, 0, 0, 0, time.UTC),
	}
	spec := specOf(t, referenceChoices())
	bp, err := Build(spec, resolved, Tools{Toolchain: pyenv.UV, NodeManager: Pnpm})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	files := map[string]golden.File{}
	for path, file := range bp.Files {
		if path == EnvFile {
			continue
		}
		files[path] = golden.File{Content: file.Content, Executable: file.Mode&0o100 != 0}
	}
	p := &Project{Cfg: spec, Resolved: resolved, Toolchain: pyenv.UV, Node: Pnpm}
	plan := p.PackagePlan()
	files[PyprojectFile] = golden.File{Content: []byte(p.RenderPyproject(appendUnique(nil, plan.Runtime...), appendUnique(nil, plan.Dev...)))}
	golden.AssertTree(t, referenceDir, files)
}

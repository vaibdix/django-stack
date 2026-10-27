package render

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var engineRoots = []string{".", filepath.Join("..", "scaffold")}

func walkEngineSources(t *testing.T, visit func(path, source string)) {
	t.Helper()
	for _, root := range engineRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && d.Name() == "testdata" {
				return filepath.SkipDir
			}
			if d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			visit(path, string(source))
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
}

func TestTheEngineNeverWarnsAndCarriesOn(t *testing.T) {
	t.Parallel()

	walkEngineSources(t, func(path, source string) {
		if strings.Contains(source, `"Warning:`) || strings.Contains(source, `"Warning: `) {
			t.Errorf("%s reports a warning and carries on; return the error instead", path)
		}
	})
}

func TestEveryCommandGoesThroughTheRunner(t *testing.T) {
	t.Parallel()

	allowed := filepath.Join("..", "scaffold", "scripts.go")
	walkEngineSources(t, func(path, source string) {
		if path != allowed && strings.Contains(source, "exec.Command") {
			t.Errorf("%s starts a process with exec.Command; use the Env's command.Runner", path)
		}
	})
}

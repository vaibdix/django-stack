package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheEngineNeverWarnsAndCarriesOn(t *testing.T) {
	t.Parallel()

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
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
		if strings.Contains(string(source), `"Warning:`) || strings.Contains(string(source), `"Warning: `) {
			t.Errorf("%s reports a warning and carries on; return the error instead", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the engine: %v", err)
	}
}

func TestEveryCommandGoesThroughTheRunner(t *testing.T) {
	t.Parallel()

	allowed := map[string]bool{
		filepath.Join("command", "runner.go"):          true,
		filepath.Join("command", "process_windows.go"): true,
		"scripts.go": true,
	}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || allowed[path] {
			return err
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(source), "exec.Command") {
			t.Errorf("%s starts a process with exec.Command; use the Env's command.Runner", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the engine: %v", err)
	}
}

func TestOnlyTheEditorsTouchTheirContent(t *testing.T) {
	t.Parallel()

	owners := map[string]bool{"settings.go": true, "pyproject.go": true}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") || owners[name] {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(source), ".content") {
			t.Errorf("%s reaches into an editor's content; add an editor method instead", name)
		}
	}
}

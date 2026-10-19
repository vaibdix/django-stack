package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheEngineNeverWarnsAndCarriesOn guards the rule that a step either
// succeeds or fails the run. "Warning:" was the marker of every path that
// reported a failure and then continued as if nothing had happened.
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

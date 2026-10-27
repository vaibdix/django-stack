package render

import (
	"context"
	"io/fs"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const goldenParseTimeout = time.Minute

func TestEveryPythonGoldenIsValidPython(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)

	var files []string
	err := filepath.WalkDir(filepath.Join("testdata", "golden"), func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ".py" {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		t.Fatalf("walking goldens: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no Python goldens found, so this test checks nothing")
	}

	ctx, cancel := context.WithTimeout(context.Background(), goldenParseTimeout)
	defer cancel()

	script := "import ast, sys\nfor path in sys.argv[1:]:\n    ast.parse(open(path).read(), path)\n"
	args := append([]string{"run", "--no-project", "--python", "3.10", "python", "-c", script}, files...)
	out, err := exec.CommandContext(ctx, uv, args...).CombinedOutput()
	if err != nil {
		t.Errorf("a golden file is not valid Python, so the golden pins broken output:\n%s", out)
	}
}

package scaffold

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

	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}

	var files []string
	err = filepath.WalkDir(filepath.Join("testdata", "golden"), func(path string, d fs.DirEntry, err error) error {
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
	out, err := exec.CommandContext(ctx, python, append([]string{"-c", script}, files...)...).CombinedOutput()
	if err != nil {
		t.Errorf("a golden file is not valid Python, so the golden pins broken output:\n%s", out)
	}
}

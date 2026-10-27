package golden

import (
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var update = flag.Bool("update-goldens", false, "rewrite golden files instead of comparing against them")

const Dir = "testdata/golden"

func Assert(t testing.TB, name, got string) {
	t.Helper()

	path := filepath.Join(Dir, name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating golden dir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("writing golden %s: %v", name, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden %s: %v\n\nRun: make goldens", name, err)
	}
	if got != string(want) {
		t.Errorf("generated output for %s does not match the golden file.\n"+
			"If this change is intentional, run:\n  make goldens\nand review the diff.\n\n%s",
			name, FirstDifference(string(want), got))
	}
}

func FirstDifference(want, got string) string {
	wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			return fmt.Sprintf("first difference at line %d:\n  want: %s\n  got:  %s", i+1, clip(w), clip(g))
		}
	}
	return "files differ only in trailing content"
}

func clip(line string) string {
	const limit = 100
	if len(line) > limit {
		line = line[:limit] + "..."
	}
	return fmt.Sprintf("%q", line)
}

type File struct {
	Content    []byte
	Executable bool
}

func AssertTree(t testing.TB, dir string, files map[string]File) {
	t.Helper()

	if *update {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("clearing %s: %v", dir, err)
		}
		for name, file := range files {
			path := filepath.Join(dir, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("creating the folder for %s: %v", name, err)
			}
			mode := os.FileMode(0o644)
			if file.Executable {
				mode = 0o755
			}
			if err := os.WriteFile(path, file.Content, mode); err != nil {
				t.Fatalf("writing %s: %v", name, err)
			}
		}
		return
	}

	found := map[string]bool{}
	tree := os.DirFS(dir)
	err := fs.WalkDir(tree, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		found[name] = true
		file, ok := files[name]
		if !ok {
			t.Errorf("%s is in %s but is no longer rendered", name, dir)
			return nil
		}
		return compareFile(t, tree, name, d, file)
	})
	if err != nil {
		t.Fatalf("reading %s: %v\n\nRun: make reference", dir, err)
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if !found[name] {
			t.Errorf("%s is rendered but missing from %s", name, dir)
		}
	}
	if t.Failed() {
		t.Log("If this change is intentional, run:\n  make reference\nand review the diff.")
	}
}

func compareFile(t testing.TB, tree fs.FS, name string, d fs.DirEntry, file File) error {
	t.Helper()
	info, err := d.Info()
	if err != nil {
		return err
	}
	if executable := info.Mode()&0o100 != 0; executable != file.Executable {
		t.Errorf("%s: executable is %v, want %v", name, executable, file.Executable)
	}
	want, err := fs.ReadFile(tree, name)
	if err != nil {
		return err
	}
	if string(want) != string(file.Content) {
		t.Errorf("%s does not match the committed copy.\n%s", name, FirstDifference(string(want), string(file.Content)))
	}
	return nil
}

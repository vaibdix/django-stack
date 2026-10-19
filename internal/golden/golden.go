// Package golden compares generated output with the files checked in under a
// package's testdata/golden directory, and rewrites them with -update.
package golden

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// update is registered once here, so every package whose tests use Assert
// accepts `go test -update` and no other package needs to declare it.
var update = flag.Bool("update", false, "rewrite golden files instead of comparing against them")

// Dir is where a package's golden files live, relative to the package.
const Dir = "testdata/golden"

// Assert fails t when got differs from the golden file name, or rewrites the
// file under -update. A missing golden file is a failure, not an empty one.
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

// FirstDifference describes the first line where want and got differ.
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

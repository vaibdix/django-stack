package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestMaskHidesValuesThatDifferOnEveryRun(t *testing.T) {
	t.Parallel()

	a := []byte("SECRET_KEY=abc\nDB_PASSWORD=one\nDB_PORT=5432\n\"createdAt\": \"2026-10-22T08:00:00Z\"\nDEBUG=True\n")
	b := []byte("SECRET_KEY=xyz\nDB_PASSWORD=two\nDB_PORT=5433\n\"createdAt\": \"2026-11-01T09:30:00Z\"\nDEBUG=True\n")
	if string(mask(a)) != string(mask(b)) {
		t.Fatalf("masked outputs differ:\n%s\n%s", mask(a), mask(b))
	}
	if c := []byte("DEBUG=False\n"); string(mask(c)) == string(mask([]byte("DEBUG=True\n"))) {
		t.Fatal("a real change was masked")
	}
}

func TestDiffProjectReportsChangedAddedAndRemovedFiles(t *testing.T) {
	t.Parallel()

	base, next := t.TempDir(), t.TempDir()
	write := func(root, rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(base, "same.py", "x = 1\n")
	write(next, "same.py", "x = 1\n")
	write(base, ".env", "SECRET_KEY=a\n")
	write(next, ".env", "SECRET_KEY=b\n")
	write(base, "changed.py", "x = 1\n")
	write(next, "changed.py", "x = 2\n")
	write(base, "gone.py", "")
	write(next, "new.py", "")
	write(next, ".venv/lib/site.py", "ignored")

	res, patch := diffProject(t.Context(), base, next)
	if !slices.Equal(res.Changed, []string{"changed.py"}) || !slices.Equal(res.OnlyBaseline, []string{"gone.py"}) || !slices.Equal(res.OnlyRun, []string{"new.py"}) {
		t.Fatalf("unexpected diff: %+v", res)
	}
	if len(patch) == 0 {
		t.Fatal("no patch for the changed file")
	}
}

func TestShimForHidesOnlyTheToolsTheProjectMustNotSee(t *testing.T) {
	t.Parallel()

	cases := []struct {
		p    project
		want string
	}{
		{project{Toolchain: "uv", Node: "pnpm", Answers: config.Answers{Stack: config.StackTailwind}}, ""},
		{project{Toolchain: "pip", Node: "pnpm", Answers: config.Answers{Stack: config.StackStandard}}, "nouv"},
		{project{Toolchain: "uv", Node: "npm", Answers: config.Answers{Stack: config.StackStandard}}, ""},
		{project{Toolchain: "pip", Node: "npm", Answers: config.Answers{Stack: config.StackTailwindHTMX}}, "nouv-nopnpm"},
	}
	for _, c := range cases {
		if got := shimFor(c.p); got != c.want {
			t.Errorf("shimFor(%s/%s/%s) = %q, want %q", c.p.Toolchain, c.p.Node, c.p.Answers.Stack, got, c.want)
		}
	}
}

func TestRunOutputInsideTheRepositoryIsRefused(t *testing.T) {
	t.Parallel()

	if err := refuseInsideRepository(t.Context(), filepath.Join("not", "yet", "created")); err == nil {
		t.Fatal("a folder inside the repository was accepted")
	}
	if err := refuseInsideRepository(t.Context(), t.TempDir()); err != nil {
		t.Fatalf("a folder outside any repository was refused: %v", err)
	}
}

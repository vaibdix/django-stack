package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/adapter"
	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/render"
	"github.com/vaibdix/django-stack/internal/report"
)

func TestTailwindScriptsCallTheLocalBinaryDirectly(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	e := &Env{reporter: report.Nop{}}
	if err := e.at(projectPath).ensurePackageJSON(); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}
	if err := e.at(projectPath).updatePackageJSONForTailwind(); err != nil {
		t.Fatalf("updatePackageJSONForTailwind: %v", err)
	}

	manifest := readFile(t, projectPath, "package.json")

	for _, forbidden := range []string{"npx ", "pnpm dlx "} {
		if strings.Contains(manifest, forbidden) {
			t.Errorf("package.json scripts still shell out via %q; the binary is a local dependency, "+
				"so both `npm run` and `pnpm run` already put it on PATH:\n%s", forbidden, manifest)
		}
	}
	if !strings.Contains(manifest, `"build:css": "tailwindcss -i`) {
		t.Errorf("build:css does not invoke the local tailwindcss binary:\n%s", manifest)
	}
}

func TestPackageJSONIsWrittenWithoutShellingOutToTheManager(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	e := &Env{reporter: report.Nop{}}

	if err := e.at(projectPath).ensurePackageJSON(); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}

	manifest := readFile(t, projectPath, "package.json")
	if !strings.Contains(manifest, `"private": true`) {
		t.Errorf("package.json is missing the private flag:\n%s", manifest)
	}
	if strings.Contains(manifest, "devEngines") {
		t.Errorf("package.json carries a devEngines block; `pnpm init` writes a range version there "+
			"that every later pnpm command rejects:\n%s", manifest)
	}
}

func TestAnExistingPackageJSONIsNeverOverwritten(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	original := `{"name":"mine","version":"9.9.9","scripts":{"test":"vitest"}}`
	writeFile(t, projectPath, "package.json", original)

	e := &Env{reporter: report.Nop{}}
	if err := e.at(projectPath).ensurePackageJSON(); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}

	if got := readFile(t, projectPath, "package.json"); got != original {
		t.Errorf("an existing package.json was rewritten:\ngot:  %s\nwant: %s", got, original)
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(data)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func TestTailwindIsRefusedUpFrontWithoutNode(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "shop", Django: "6.1.2", Database: config.SQLite, Stack: config.StackTailwind, App: "catalog"}
	python := config.Resolved{PythonPath: "/usr/bin/python3"}

	missing := func(context.Context, command.Runner) (render.NodeManager, bool) { return render.NodeManager{}, false }
	if _, err := preflight(context.Background(), specOf(t, cfg), python, adapter.Complete(port.Ports{Run: command.NewOS()}), deps{nodeManager: missing}.withDefaults()); err == nil || !strings.Contains(err.Error(), "Node.js") {
		t.Errorf("a Tailwind project without Node would ship a page referencing CSS that is never built; preflight() = %v", err)
	}

	present := func(context.Context, command.Runner) (render.NodeManager, bool) { return render.Pnpm, true }
	tools, err := preflight(context.Background(), specOf(t, cfg), python, adapter.Complete(port.Ports{Run: command.NewOS()}), deps{nodeManager: present}.withDefaults())
	if err != nil {
		t.Errorf("preflight() with Node available = %v", err)
	}
	if tools.NodeManager.Name != render.Pnpm.Name {
		t.Errorf("preflight() decided %q, want pnpm", tools.NodeManager.Name)
	}
}

func TestPackageJSONWritesTheNodeRangeLiterally(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	e := &Env{reporter: report.Nop{}}
	if err := e.at(projectPath).ensurePackageJSON(); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}
	if err := e.at(projectPath).updatePackageJSONForTailwind(); err != nil {
		t.Fatalf("updatePackageJSONForTailwind: %v", err)
	}

	manifest := readFile(t, projectPath, "package.json")
	want := `"node": ">=` + strconv.Itoa(render.TailwindMinNode()) + `"`
	if !strings.Contains(manifest, want) {
		t.Errorf("package.json does not contain %s:\n%s", want, manifest)
	}
	if !strings.HasSuffix(manifest, "}\n") || strings.HasSuffix(manifest, "}\n\n") {
		t.Errorf("package.json must end with exactly one newline:\n%q", manifest)
	}
}

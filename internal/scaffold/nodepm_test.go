package scaffold

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func TestPnpmIsPreferredAndNpmIsTheFallback(t *testing.T) {
	t.Parallel()

	if PNPM.Name != "pnpm" || NPM.Name != "npm" {
		t.Fatalf("unexpected manager names: %q and %q", PNPM.Name, NPM.Name)
	}

	name, args := PNPM.Install("tailwindcss")
	if name != "pnpm" || args[0] != "add" {
		t.Errorf("pnpm install = %s %v, want pnpm add", name, args)
	}

	name, args = NPM.Install("tailwindcss")
	if name != "npm" || args[0] != "install" {
		t.Errorf("npm install = %s %v, want npm install", name, args)
	}
}

func TestInstallSkipsDependencyBuildScripts(t *testing.T) {
	t.Parallel()

	for _, manager := range []NodeManager{PNPM, NPM} {
		_, args := manager.Install("tailwindcss", "@tailwindcss/cli")
		if !contains(args, "--ignore-scripts") {
			t.Errorf("%s install does not pass --ignore-scripts: %v; "+
				"pnpm exits non-zero on unapproved dependency build scripts, which aborts Tailwind setup",
				manager.Name, args)
		}
		if !contains(args, "tailwindcss") || !contains(args, "@tailwindcss/cli") {
			t.Errorf("%s install dropped a package: %v", manager.Name, args)
		}
	}
}

func TestRunCommandNamesTheManagerThatWasFound(t *testing.T) {
	t.Parallel()

	if got := PNPM.RunCommand("watch:css"); got != "pnpm run watch:css" {
		t.Errorf("RunCommand = %q", got)
	}
	if got := NPM.RunCommand("watch:css"); got != "npm run watch:css" {
		t.Errorf("RunCommand = %q", got)
	}
}

func TestTailwindScriptsCallTheLocalBinaryDirectly(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	e := &Env{Reporter: NopReporter{}}
	if err := e.ensurePackageJSON(projectPath); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}
	if err := e.updatePackageJSONForTailwind(projectPath); err != nil {
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
	e := &Env{Reporter: NopReporter{}}

	if err := e.ensurePackageJSON(projectPath); err != nil {
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

	e := &Env{Reporter: NopReporter{}}
	if err := e.ensurePackageJSON(projectPath); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}

	if got := readFile(t, projectPath, "package.json"); got != original {
		t.Errorf("an existing package.json was rewritten:\ngot:  %s\nwant: %s", got, original)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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

func TestGeneratedFilesNeverContainAFormattingError(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	appPath := filepath.Join(projectPath, "catalog")
	if err := os.MkdirAll(appPath, 0o755); err != nil {
		t.Fatalf("creating the app directory: %v", err)
	}

	e := &Env{
		Cfg:      config.Project{ProjectName: "demo", AppName: "catalog", CreateAppTemplates: true},
		Reporter: NopReporter{},
	}
	if err := e.writeBookApp(projectPath); err != nil {
		t.Fatalf("writeBookApp: %v", err)
	}

	err := filepath.WalkDir(projectPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), "%!") {
			rel, _ := filepath.Rel(projectPath, path)
			t.Errorf("%s carries a Go formatting error, so the generated file is not valid source:\n%s",
				rel, data)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the generated project: %v", err)
	}
}

func TestTailwindIsRefusedUpFrontWithoutNode(t *testing.T) {
	t.Parallel()

	cfg := config.Project{ProjectName: "shop", SelectedPythonPath: "/usr/bin/python3", DjangoVersion: "6.1.2", SetupTailwind: true}

	missing := &Env{Cfg: cfg, nodeManagerLookup: func() (NodeManager, bool) { return NodeManager{}, false }}
	if err := missing.validate(context.Background()); err == nil || !strings.Contains(err.Error(), "Node.js") {
		t.Errorf("a Tailwind project without Node would ship a page referencing CSS that is never built; validate() = %v", err)
	}

	present := &Env{Cfg: cfg, nodeManagerLookup: func() (NodeManager, bool) { return PNPM, true }}
	if err := present.validate(context.Background()); err != nil {
		t.Errorf("validate() with Node available = %v", err)
	}
}

func TestThePinnedTailwindReleaseExistsUpstream(t *testing.T) {
	if os.Getenv("DJANGO_STACK_NETWORK_TESTS") == "" {
		t.Skip("set DJANGO_STACK_NETWORK_TESTS=1 to check the pinned Tailwind release against npm")
	}
	for _, pkg := range []string{"tailwindcss", "@tailwindcss/cli"} {
		out, err := exec.CommandContext(t.Context(), "npm", "view", pkg+"@"+TailwindVersion, "version").Output()
		if err != nil || strings.TrimSpace(string(out)) != TailwindVersion {
			t.Errorf("npm has no %s@%s (err=%v, got %q); bump TailwindVersion", pkg, TailwindVersion, err, out)
		}
	}
}

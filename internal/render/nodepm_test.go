package render

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/livetest"
)

func TestPnpmIsPreferredAndNpmIsTheFallback(t *testing.T) {
	t.Parallel()

	if Pnpm.Name != "pnpm" || NPM.Name != "npm" {
		t.Fatalf("unexpected manager names: %q and %q", Pnpm.Name, NPM.Name)
	}

	name, args := Pnpm.Install("tailwindcss")
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

	for _, manager := range []NodeManager{Pnpm, NPM} {
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

	if got := Pnpm.RunCommand("watch:css"); got != "pnpm run watch:css" {
		t.Errorf("RunCommand = %q", got)
	}
	if got := NPM.RunCommand("watch:css"); got != "npm run watch:css" {
		t.Errorf("RunCommand = %q", got)
	}
}

func contains(values []string, want string) bool {
	return slices.Contains(values, want)
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(data)
}

func TestGeneratedFilesNeverContainAFormattingError(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	appPath := filepath.Join(projectPath, "catalog")
	if err := os.MkdirAll(appPath, 0o755); err != nil {
		t.Fatalf("creating the app directory: %v", err)
	}

	e := &Project{
		Cfg: mustSpec(config.Choices{Name: "demo", App: "catalog", Stack: config.StackStandard}),
	}
	if err := writeBookApp(e, projectPath); err != nil {
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

func TestThePinnedTailwindReleaseExistsUpstream(t *testing.T) {
	t.Parallel()

	livetest.Require(t, "check the pinned Tailwind release against npm")
	for _, pkg := range []string{"tailwindcss", "@tailwindcss/cli"} {
		out, err := exec.CommandContext(t.Context(), "npm", "view", pkg+"@"+TailwindVersion(), "version").Output()
		if err != nil || strings.TrimSpace(string(out)) != TailwindVersion() {
			t.Errorf("npm has no %s@%s (err=%v, got %q); bump TailwindVersion", pkg, TailwindVersion(), err, out)
		}
	}
}

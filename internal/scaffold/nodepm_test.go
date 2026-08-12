package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func withNodeManager(t *testing.T, manager NodeManager, found bool) {
	t.Helper()

	original := detectNodeManager
	detectNodeManager = func() (NodeManager, bool) { return manager, found }
	t.Cleanup(func() { detectNodeManager = original })
}

func TestPnpmIsPreferredAndNpmIsTheFallback(t *testing.T) {
	t.Parallel()

	if pnpmManager.Name != "pnpm" || npmManager.Name != "npm" {
		t.Fatalf("unexpected manager names: %q and %q", pnpmManager.Name, npmManager.Name)
	}

	name, args := pnpmManager.Install("tailwindcss")
	if name != "pnpm" || args[0] != "add" {
		t.Errorf("pnpm install = %s %v, want pnpm add", name, args)
	}

	name, args = npmManager.Install("tailwindcss")
	if name != "npm" || args[0] != "install" {
		t.Errorf("npm install = %s %v, want npm install", name, args)
	}
}

func TestInstallSkipsDependencyBuildScripts(t *testing.T) {
	t.Parallel()

	for _, manager := range []NodeManager{pnpmManager, npmManager} {
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

func TestDlxMapsToPnpmDlxAndNpx(t *testing.T) {
	t.Parallel()

	name, args := pnpmManager.Dlx("tailwindcss", "--help")
	if name != "pnpm" || args[0] != "dlx" || args[1] != "tailwindcss" || args[2] != "--help" {
		t.Errorf("pnpm one-off = %s %v, want pnpm dlx tailwindcss --help", name, args)
	}

	name, args = npmManager.Dlx("tailwindcss", "--help")
	if name != "npx" || args[0] != "tailwindcss" || args[1] != "--help" {
		t.Errorf("npm one-off = %s %v, want npx tailwindcss --help", name, args)
	}

	if got := pnpmManager.DlxCommand("tailwindcss"); got != "pnpm dlx tailwindcss" {
		t.Errorf("DlxCommand = %q, want %q", got, "pnpm dlx tailwindcss")
	}
	if got := npmManager.DlxCommand("tailwindcss"); got != "npx tailwindcss" {
		t.Errorf("DlxCommand = %q, want %q", got, "npx tailwindcss")
	}
}

func (m NodeManager) Dlx(command string, args ...string) (string, []string) {
	full := append([]string{}, m.dlx[1:]...)
	full = append(full, command)
	return m.dlx[0], append(full, args...)
}

func (m NodeManager) DlxCommand(command string) string {
	name, args := m.Dlx(command)
	for _, arg := range args {
		name += " " + arg
	}
	return name
}

func TestRunCommandNamesTheManagerThatWasFound(t *testing.T) {
	t.Parallel()

	if got := pnpmManager.RunCommand("watch:css"); got != "pnpm run watch:css" {
		t.Errorf("RunCommand = %q", got)
	}
	if got := npmManager.RunCommand("watch:css"); got != "npm run watch:css" {
		t.Errorf("RunCommand = %q", got)
	}
}

func TestTailwindScriptsCallTheLocalBinaryDirectly(t *testing.T) {
	t.Parallel()

	withNodeManager(t, pnpmManager, true)

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
	if err := e.applyAppTemplates(projectPath); err != nil {
		t.Fatalf("applyAppTemplates: %v", err)
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

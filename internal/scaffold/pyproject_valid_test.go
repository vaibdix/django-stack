package scaffold

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
)

func TestEveryGeneratedPyprojectIsValidToml(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)

	for _, tc := range []struct {
		name string
		cfg  config.Project
	}{
		{
			name: "everything on",
			cfg: config.Project{
				ProjectName: "demo", AppName: "catalog", DjangoVersion: "6.1",
				SelectedPythonVersion: "Python 3.13.2", DatabaseType: config.PostgresDocker,
				SetupRuff: true, SetupMypy: true, SetupRestFramework: true,
				SetupPydanticConf: true, SetupMultiEnv: true,
			},
		},
		{
			name: "ruff only",
			cfg: config.Project{
				ProjectName: "demo", DjangoVersion: "6.1",
				SelectedPythonVersion: "Python 3.13.2", DatabaseType: config.SQLite,
				SetupRuff: true,
			},
		},
		{
			name: "mypy only",
			cfg: config.Project{
				ProjectName: "demo", DjangoVersion: "6.1",
				SelectedPythonVersion: "Python 3.13.2", DatabaseType: config.SQLite,
				SetupMypy: true,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := writeToolingConfig(t, tc.cfg)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			t.Cleanup(cancel)

			const script = `
import sys, tomllib
with open('pyproject.toml', 'rb') as handle:
    tomllib.load(handle)
print('OK')
`
			cmd := exec.CommandContext(ctx, uv, "run", "--no-project", "python", "-c", script)
			cmd.Dir = root

			out, runErr := cmd.CombinedOutput()
			if runErr != nil || !strings.Contains(string(out), "OK") {
				t.Fatalf("the generated pyproject.toml is not parseable, so every tool that reads it fails:\n%s", out)
			}
		})
	}
}

func writeToolingConfig(t *testing.T, cfg config.Project) string {
	t.Helper()

	root := t.TempDir()
	e := &Env{Cfg: cfg, Reporter: NopReporter{}, Runner: commandtest.NewFake()}

	moduleDir := filepath.Join(root, cfg.ModuleName())
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("creating the module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, SettingsFile), []byte(djangoAdminSettingsFixture), 0o644); err != nil {
		t.Fatalf("writing the settings fixture: %v", err)
	}

	if cfg.SetupRuff {
		if err := e.applyRuff(root); err != nil {
			t.Fatalf("applyRuff: %v", err)
		}
	}
	if cfg.SetupMypy {
		if err := e.applyMypy(root); err != nil {
			t.Fatalf("applyMypy: %v", err)
		}
	}

	return root
}

func TestPerFileIgnoresDeclareEachPatternOnce(t *testing.T) {
	t.Parallel()

	written := ruffConfigFor(t, config.Project{
		ProjectName: "demo", DjangoVersion: "6.1",
		SelectedPythonVersion: "Python 3.13.2", SetupRuff: true,
	})

	start := strings.Index(written, "[tool.ruff.lint.per-file-ignores]")
	if start == -1 {
		t.Fatalf("the generated config declares no per-file ignores:\n%s", written)
	}
	section := written[start:]
	if end := strings.Index(section[1:], "\n["); end != -1 {
		section = section[:end+1]
	}

	seen := make(map[string]int)
	for _, line := range strings.Split(section, "\n") {
		key, _, ok := strings.Cut(strings.TrimSpace(line), " = ")
		if !ok || !strings.HasPrefix(key, `"`) {
			continue
		}
		seen[key]++
	}

	for key, count := range seen {
		if count > 1 {
			t.Errorf("%s is declared %d times; a duplicate key makes the whole file invalid TOML", key, count)
		}
	}
	if len(seen) == 0 {
		t.Error("no per-file ignores were found, so this test proves nothing")
	}
}

func ruffConfigFor(t *testing.T, cfg config.Project) string {
	t.Helper()

	root := writeToolingConfig(t, cfg)
	written, err := os.ReadFile(filepath.Join(root, PyprojectFile))
	if err != nil {
		t.Fatalf("reading pyproject.toml: %v", err)
	}
	return string(written)
}

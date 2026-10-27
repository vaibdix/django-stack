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
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/report"
)

func TestEveryGeneratedPyprojectIsValidToml(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)

	for _, tc := range []struct {
		name string
		cfg  config.Choices
	}{
		{
			name: "everything on",
			cfg: config.Choices{
				Name:     "demo",
				App:      "catalog",
				Django:   "6.1",
				Database: config.PostgresDocker,
				AddOns:   config.NewSet(config.RestFramework, config.Ruff, config.Mypy, config.MultiEnv, config.Pydantic),
			},
		},
		{
			name: "ruff only",
			cfg: config.Choices{
				Name:     "demo",
				Django:   "6.1",
				Database: config.SQLite,
				AddOns:   config.NewSet(config.Ruff),
			},
		},
		{
			name: "mypy only",
			cfg: config.Choices{
				Name:     "demo",
				Django:   "6.1",
				Database: config.SQLite,
				AddOns:   config.NewSet(config.Mypy),
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
with open("pyproject.toml", "rb") as handle:
    tomllib.load(handle)
print("OK")
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

func writeToolingConfig(t *testing.T, cfg config.Choices) string {
	t.Helper()

	root := t.TempDir()
	e := &Env{cfg: mustSpec(cfg), resolved: python313, dir: root, reporter: report.Nop{}, ports: port.Ports{Run: commandtest.NewFake()}}

	moduleDir := filepath.Join(root, cfg.ModuleName())
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("creating the module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, settingsFile), []byte(djangoAdminSettingsFixture), 0o644); err != nil {
		t.Fatalf("writing the settings fixture: %v", err)
	}

	if cfg.Has(config.Ruff) {
		if err := e.writePyprojectWith(root, config.Ruff); err != nil {
			t.Fatalf("applyRuff: %v", err)
		}
	}
	if cfg.Has(config.Mypy) {
		if err := e.writePyprojectWith(root, config.Mypy); err != nil {
			t.Fatalf("applyMypy: %v", err)
		}
	}

	return root
}

func TestPerFileIgnoresDeclareEachPatternOnce(t *testing.T) {
	t.Parallel()

	written := ruffConfigFor(t, config.Choices{
		Name:   "demo",
		Django: "6.1",
		AddOns: config.NewSet(config.Ruff),
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
	for line := range strings.SplitSeq(section, "\n") {
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

var python313 = config.Resolved{PythonVersion: "Python 3.13.2"}

func ruffConfigFor(t *testing.T, cfg config.Choices) string {
	t.Helper()

	root := writeToolingConfig(t, cfg)
	written, err := os.ReadFile(filepath.Join(root, pyprojectFile))
	if err != nil {
		t.Fatalf("reading pyproject.toml: %v", err)
	}
	return string(written)
}

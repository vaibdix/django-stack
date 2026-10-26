package scaffold

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

func TestRenderedSettingsPassTheDeployCheckOnEverySeries(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)
	for series, version := range map[string]string{"5.2": "5.2.18", "6.0": "6.0.9", "6.1": "6.1.2"} {
		t.Run(series, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			cfg := config.Choices{Name: "deploy-check", Django: version, Database: config.SQLite, Stack: config.StackStandard, App: "catalog"}
			resolved := config.Resolved{PythonVersion: "Python 3.13.2", SecretKey: strings.Repeat("k3y-", 16)}
			bp, err := Build(cfg, resolved, Tools{Toolchain: pyenv.UV})
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			for path, file := range bp.Files {
				target := filepath.Join(root, filepath.FromSlash(path))
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, file.Content, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			ctx, cancel := context.WithTimeout(t.Context(), confRuntimeTimeout)
			defer cancel()
			args := []string{"run", "--no-project", "--python", "3.13", "--with", "django==" + version,
				"--with", pkgEnviron, "--with", pkgWhiteNoise, "--with", pkgArgon2, "--with", pkgBrowserReload}
			if !config.DjangoSupportsNativeCSP(version) {
				args = append(args, "--with", pkgDjangoCSP)
			}
			args = append(args, "python", "manage.py", "check", "--deploy", "--fail-level", "WARNING")
			cmd := exec.CommandContext(ctx, uv, args...)
			cmd.Dir = root
			cmd.Env = append(os.Environ(),
				"ENVIRONMENT=production", "SECRET_KEY="+resolved.SecretKey, "ALLOWED_HOSTS=example.com",
				"DEFAULT_FROM_EMAIL=ops@example.com", "SECURE_HSTS_INCLUDE_SUBDOMAINS=true", "SECURE_HSTS_PRELOAD=true")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("manage.py check --deploy on Django %s: %v\n%s", version, err, out)
			}
		})
	}
}

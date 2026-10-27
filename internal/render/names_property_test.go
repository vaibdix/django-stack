package render

import (
	"context"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

func randomAppNames(seed uint64, lengths ...int) []string {
	r := rand.New(rand.NewPCG(seed, seed))
	const first, rest = "abcdefghijklmnopqrstuvwxyz", "abcdefghijklmnopqrstuvwxyz0123456789_"
	names := make([]string, 0, len(lengths))
	for _, n := range lengths {
		name := []byte{first[r.IntN(len(first))]}
		for len(name) < n {
			name = append(name, rest[r.IntN(len(rest))])
		}
		names = append(names, string(name))
	}
	return names
}

func TestRenderedMigrationsMatchDjangoForAnyAppName(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)
	for _, app := range randomAppNames(11, 1, 6, 17, 30) {
		t.Run(app, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			cfg := config.Choices{Name: "index-names", Django: "6.1.2", Database: config.SQLite, Stack: config.StackStandard, App: app}
			resolved := config.Resolved{PythonVersion: "Python 3.13.2", SecretKey: "index-names-only-0123456789-abcdefghijklmnopqrstuvwxyz"}
			bp, err := Build(specOf(t, cfg), resolved, Tools{Toolchain: pyenv.UV})
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
			cmd := exec.CommandContext(ctx, uv, "run", "--no-project", "--python", "3.13", "--with", "django==6.1.2",
				"--with", PkgEnviron, "--with", pkgWhiteNoise, "--with", pkgArgon2, "--with", pkgBrowserReload,
				"python", "manage.py", "makemigrations", "--check", "--dry-run")
			cmd.Dir = root
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("Django would write different migrations for app %q, so the index names the scaffold "+
					"computes have drifted from Django's: %v\n%s", app, err, out)
			}
		})
	}
}

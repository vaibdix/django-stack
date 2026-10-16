package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

type contextRecorder struct {
	mu   sync.Mutex
	live []string
}

func (r *contextRecorder) Run(ctx context.Context, _, name string, args ...string) ([]byte, error) {
	if ctx.Err() == nil {
		r.mu.Lock()
		r.live = append(r.live, name+" "+filepath.Base(firstOr(args, "")))
		r.mu.Unlock()
	}
	return nil, nil
}

func firstOr(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}

func TestEveryCommandStepHonoursCancellation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	moduleDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("creating the module: %v", err)
	}
	for path, content := range map[string]string{
		filepath.Join(root, ManageFile):            "def main():\n    pass\n",
		filepath.Join(moduleDir, SettingsFile):     djangoAdminSettingsFixture,
		filepath.Join(root, "catalog", "__init__"): "",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	recorder := &contextRecorder{}
	e := &Env{
		Cfg:      config.Project{ProjectName: "demo", AppName: "catalog", InitializeGit: true, SetupPreCommit: true},
		Root:     root,
		Runner:   recorder,
		Reporter: NopReporter{},
		ctx:      ctx,
	}

	steps := map[string]func(string) error{
		"create django project": e.createDjangoProject,
		"initialize git":        e.initGitRepository,
		"run migrations":        e.runMigrations,
		"setup pre-commit":      e.applyPreCommit,
		"create django app": func(p string) error {
			return e.createDjangoApp(p, filepath.Join(moduleDir, SettingsFile))
		},
	}
	for name, step := range steps {
		_ = step(root)
		if len(recorder.live) > 0 {
			t.Errorf("%s ran %v with a live context after the run was cancelled", name, recorder.live)
			recorder.live = nil
		}
	}
}

package scaffold

import (
	"context"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/report"
)

const startprojectTimeout = 3 * time.Minute

func TestTheSettingsEditsApplyToEverySupportedDjango(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)

	for _, rule := range config.DjangoCompatibilityRules {
		t.Run(rule.Series, func(t *testing.T) {
			t.Parallel()

			for name, cfg := range startprojectConfigs(rule.Series) {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					root := startproject(t, uv, rule.Series, cfg.ModuleName())
					applySettingsEdits(t, root, cfg)
					parseEveryPythonFile(t, uv, root)
				})
			}
		})
	}
}

func startprojectConfigs(series string) map[string]config.Choices {
	version := series + ".0"

	standard := config.Choices{Name: "proj", Django: version, Database: config.SQLite}
	standard.Stack, standard.AddOns = config.StackStandard, nil

	typedSplit := config.Choices{Name: "proj", Django: version, Database: config.PostgresDocker}
	typedSplit.Stack, typedSplit.AddOns = config.StackHTMX, config.NewSet(config.Pydantic, config.MultiEnv)

	apiOnly := config.Choices{Name: "proj", Django: version, Database: config.MySQLDocker}
	apiOnly.Stack, apiOnly.AddOns = config.StackAPIOnly, config.NewSet(config.MultiEnv)

	return map[string]config.Choices{"standard": standard, "typed-split": typedSplit, "api-only-split": apiOnly}
}

func startproject(t *testing.T, uv, series, module string) string {
	t.Helper()

	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), startprojectTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, uv, "run", "--no-project", "--python", "3.13", "--with", "django=="+series+".*",
		"django-admin", "startproject", module, root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("django-admin startproject (Django %s): %v\n%s", series, err, out)
	}
	return root
}

func applySettingsEdits(t *testing.T, root string, cfg config.Choices) {
	t.Helper()

	e := &Env{cfg: cfg, resolved: credentialsFor(cfg.Database, config.DatabaseCredentials{}), reporter: report.Nop{}}
	steps := []struct {
		name string
		run  func() error
	}{
		{"configure settings", func() error { return e.configureProjectSettings(root) }},
	}
	if cfg.CreateTemplates() {
		steps = append(steps, struct {
			name string
			run  func() error
		}{"templates and static files", func() error {
			se, err := newSettingsEditor(e.settingsPath(root))
			if err != nil {
				return err
			}
			if err := se.ConfigureTemplatesDirs(); err != nil {
				return err
			}
			return se.Save()
		}})
	}
	if cfg.Has(config.MultiEnv) {
		steps = append(steps, struct {
			name string
			run  func() error
		}{"split settings", func() error { return e.applyMultiEnvSettings(root) }})
	}

	for _, step := range steps {
		if err := step.run(); err != nil {
			t.Fatalf("%s on Django %s's startproject output: %v", step.name, cfg.Django, err)
		}
	}
}

func parseEveryPythonFile(t *testing.T, uv, root string) {
	t.Helper()

	var files []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".py") {
			files = append(files, path)
		}
		return err
	}); err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), startprojectTimeout)
	defer cancel()
	script := "import ast, sys\nfor path in sys.argv[1:]:\n    ast.parse(open(path).read(), path)\n"
	args := append([]string{"run", "--no-project", "--python", "3.13", "python", "-c", script}, files...)
	if out, err := exec.CommandContext(ctx, uv, args...).CombinedOutput(); err != nil {
		t.Errorf("the edited project is not valid Python:\n%s", out)
	}
}

package scaffold

import (
	"context"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
)

const startprojectTimeout = 3 * time.Minute

// TestTheSettingsEditsApplyToEverySupportedDjango runs the real
// `django-admin startproject` of each series the generator offers and applies
// the settings edits to its output. The edits anchor on Django's own text, so
// this is the test that fails the day a new series changes that text, instead
// of a user's run failing with an AnchorError.
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

func startprojectConfigs(series string) map[string]config.Project {
	version := series + ".0"

	standard := config.Project{ProjectName: "proj", DjangoVersion: version, DatabaseType: config.SQLite}
	standard.ApplyFeatures(config.StackStandard.Features())

	typedSplit := config.Project{ProjectName: "proj", DjangoVersion: version, DatabaseType: config.PostgresDocker}
	typedSplit.ApplyFeatures(append(config.StackHTMX.Features(), config.FeaturePydanticConfig, config.FeatureMultiEnv))

	apiOnly := config.Project{ProjectName: "proj", DjangoVersion: version, DatabaseType: config.MySQLDocker}
	apiOnly.ApplyFeatures([]config.Feature{config.FeatureMultiEnv})

	return map[string]config.Project{"standard": standard, "typed-split": typedSplit, "api-only-split": apiOnly}
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

func applySettingsEdits(t *testing.T, root string, cfg config.Project) {
	t.Helper()

	e := &Env{Cfg: withCredentials(cfg), Reporter: NopReporter{}}
	steps := []struct {
		name string
		run  func() error
	}{
		{"strip comments", func() error { return e.stripCommentsFromDjangoBoilerplate(root) }},
		{"configure settings", func() error { return e.configureProjectSettings(root) }},
	}
	if cfg.CreateTemplates {
		steps = append(steps, struct {
			name string
			run  func() error
		}{"templates and static files", func() error {
			se, err := NewSettingsEditor(e.settingsPath(root))
			if err != nil {
				return err
			}
			if err := se.ConfigureTemplatesDirs(); err != nil {
				return err
			}
			return se.Save()
		}})
	}
	if cfg.SetupMultiEnv {
		steps = append(steps, struct {
			name string
			run  func() error
		}{"split settings", func() error { return e.applyMultiEnvSettings(root) }})
	}

	for _, step := range steps {
		if err := step.run(); err != nil {
			t.Fatalf("%s on Django %s's startproject output: %v", step.name, cfg.DjangoVersion, err)
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

package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/golden"

	"github.com/vaibdix/django-cli/internal/config"
)

var settingsFixtures = map[string]string{"5.2": "5.2.18", "6.0": "6.0.9", "6.1": "6.1.2"}

func newSettingsFixture(t *testing.T) string {
	t.Helper()
	return newSettingsFixtureFor(t, "5.2")
}

func newSettingsFixtureFor(t *testing.T, series string) string {
	t.Helper()

	original, err := os.ReadFile(filepath.Join("testdata", "django_settings_"+series+".py"))
	if err != nil {
		t.Fatalf("reading Django settings fixture: %v", err)
	}

	path := filepath.Join(t.TempDir(), "settings.py")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("writing settings fixture copy: %v", err)
	}
	return path
}

func TestGoldenConfigureDjangoSettings(t *testing.T) {
	t.Parallel()

	databases := []config.Database{config.SQLite, config.PostgresDocker, config.MySQLDocker}

	for series, release := range settingsFixtures {
		for _, db := range databases {
			t.Run(series+"/"+string(db), func(t *testing.T) {
				path := newSettingsFixtureFor(t, series)

				e := &Env{Cfg: config.Project{
					ProjectName:     "my-shop",
					DatabaseType:    db,
					DjangoVersion:   release,
					CreateTemplates: true,
				}}

				if err := e.configureSettings(path); err != nil {
					t.Fatalf("configureSettings: %v", err)
				}

				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("reading configured settings: %v", err)
				}
				golden.Assert(t, "settings/"+series+"/configured_"+string(db)+".py", string(got))
			})
		}
	}
}

func TestConfigureDjangoSettingsAppliesEveryAnchoredEdit(t *testing.T) {
	t.Parallel()

	path := newSettingsFixture(t)

	e := &Env{Cfg: config.Project{
		ProjectName:     "my-shop",
		DatabaseType:    config.PostgresDocker,
		CreateTemplates: true,
	}}

	if err := e.configureSettings(path); err != nil {
		t.Fatalf("configureSettings: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading configured settings: %v", err)
	}
	got := string(raw)

	edits := map[string]string{
		"django-environ import":     "import environ",
		"browser-reload app":        AppBrowserReload,
		"browser-reload middleware": MiddlewareBrowserReload,
		"project context processor": "my_shop.context_processors",
		"security settings block":   "PASSWORD_HASHERS",
		"env-driven SECRET_KEY":     "env(",
		"postgres database backend": "postgresql",
	}

	for name, needle := range edits {
		if !strings.Contains(got, needle) {
			t.Errorf("anchored edit %q did not apply: %q missing from the configured settings.py.\n"+
				"This is the silent-no-op failure mode — the edit found no anchor and returned without error.",
				name, needle)
		}
	}

	if strings.Contains(got, "django.db.backends.sqlite3") {
		t.Error("the stock sqlite3 DATABASES block survived postgres configuration")
	}
}

func TestProductionErrorsReachTheConsoleWithoutStructlog(t *testing.T) {
	t.Parallel()

	for _, structlog := range []bool{false, true} {
		path := newSettingsFixture(t)
		e := &Env{Cfg: config.Project{ProjectName: "my-shop", DatabaseType: config.SQLite, SetupStructlog: structlog}}
		if err := e.configureSettings(path); err != nil {
			t.Fatalf("configureSettings: %v", err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading configured settings: %v", err)
		}
		got := string(raw)

		if structlog {
			if strings.Contains(got, "LOGGING = ") {
				t.Errorf("structlog writes its own LOGGING; a second one would shadow it:\n%s", got)
			}
			continue
		}
		if !strings.Contains(got, consoleLoggingBlock) {
			t.Errorf("without structlog Django's default LOGGING drops production errors "+
				"(console needs DEBUG, mail_admins needs ADMINS); expected the console block:\n%s", got)
		}
	}
}

func TestSingleFileSettingsBranchOnDebugOnce(t *testing.T) {
	t.Parallel()

	path := newSettingsFixture(t)
	e := &Env{Cfg: config.Project{
		ProjectName: "my-shop", DatabaseType: config.SQLite, DjangoVersion: "6.1.2", CreateTemplates: true,
	}}
	if err := e.configureSettings(path); err != nil {
		t.Fatalf("configureSettings: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading configured settings: %v", err)
	}
	settings := string(raw)

	if n := strings.Count(settings, "\nif DEBUG:"); n != 1 || strings.Contains(settings, "if not DEBUG") {
		t.Errorf("development and production settings are scattered over several DEBUG branches:\n%s", settings)
	}
	for _, want := range []string{
		manifestStaticStorage,
		"'django_cache_table'",
		"CSRF_TRUSTED_ORIGINS = env.list('CSRF_TRUSTED_ORIGINS', default=[])",
		placeholderSecretKey,
		"'django_browser_reload'",
		"'django.core.mail.backends.smtp.EmailBackend'",
	} {
		if !strings.Contains(settings, want) {
			t.Errorf("single-file settings lost %q:\n%s", want, settings)
		}
	}
}

func TestAPIOnlyProjectsCanCollectStatic(t *testing.T) {
	t.Parallel()

	path := newSettingsFixture(t)
	e := &Env{Cfg: config.Project{ProjectName: "my-shop", DatabaseType: config.SQLite}}
	if err := e.configureSettings(path); err != nil {
		t.Fatalf("configureSettings: %v", err)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "STATIC_ROOT = BASE_DIR / 'staticfiles'") {
		t.Errorf("without STATIC_ROOT, collectstatic refuses to run and the admin has no static files in production:\n%s", raw)
	}
}

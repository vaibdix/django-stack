package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func newSettingsFixture(t *testing.T) string {
	t.Helper()

	original, err := os.ReadFile(filepath.Join("testdata", "django_settings_5.2.py"))
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

	for _, db := range databases {
		t.Run(string(db), func(t *testing.T) {
			path := newSettingsFixture(t)

			e := &Env{Cfg: config.Project{
				ProjectName:   "my-shop",
				DatabaseType:  db,
				DjangoVersion: "6.1",
			}}

			if err := e.configureSettings(path); err != nil {
				t.Fatalf("configureSettings: %v", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading configured settings: %v", err)
			}
			assertGolden(t, "settings/configured_"+string(db)+".py", string(got))
		})
	}
}

func TestConfigureDjangoSettingsIsIdempotent(t *testing.T) {
	t.Parallel()

	path := newSettingsFixture(t)

	e := &Env{Cfg: config.Project{
		ProjectName:  "my-shop",
		DatabaseType: config.PostgresDocker,
	}}

	if err := e.configureSettings(path); err != nil {
		t.Fatalf("first configureSettings: %v", err)
	}
	afterFirst, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading after first pass: %v", err)
	}

	if err := e.configureSettings(path); err != nil {
		t.Fatalf("second configureSettings: %v", err)
	}
	afterSecond, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading after second pass: %v", err)
	}

	if string(afterFirst) != string(afterSecond) {
		t.Errorf("configureSettings is not idempotent.\n%s",
			firstDifference(string(afterFirst), string(afterSecond)))
	}
}

func TestConfigureDjangoSettingsAppliesEveryAnchoredEdit(t *testing.T) {
	t.Parallel()

	path := newSettingsFixture(t)

	e := &Env{Cfg: config.Project{
		ProjectName:  "my-shop",
		DatabaseType: config.PostgresDocker,
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

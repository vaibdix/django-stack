package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func TestSettingsEditorAddToListInsertsItem(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: `INSTALLED_APPS = [
    'django.contrib.admin',
]`}

	if err := se.AddToList("INSTALLED_APPS", "rest_framework"); err != nil {
		t.Fatalf("AddToList failed: %v", err)
	}

	if !strings.Contains(se.Content(), "'rest_framework'") {
		t.Fatalf("expected rest_framework in content, got:\n%s", se.Content())
	}
}

func TestSettingsEditorAddToListIsIdempotent(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: `INSTALLED_APPS = [
    'django.contrib.admin',
]`}

	_ = se.AddToList("INSTALLED_APPS", "rest_framework")
	first := se.Content()
	_ = se.AddToList("INSTALLED_APPS", "rest_framework")

	if se.Content() != first {
		t.Fatalf("AddToList should be idempotent")
	}
}

func TestSettingsEditorAddToListMissingListReturnsError(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: "SECRET_KEY = 'abc'"}

	err := se.AddToList("INSTALLED_APPS", "rest_framework")
	if err == nil {
		t.Fatalf("expected error when list is missing")
	}
}

func TestSettingsEditorAppendBlock(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: "BASE = True\n"}
	se.AppendBlock("\nEXTRA = False\n")

	if !strings.Contains(se.Content(), "EXTRA = False") {
		t.Fatalf("AppendBlock did not append, got:\n%s", se.Content())
	}
}

func TestSettingsEditorRemoveCoreSetting(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: `from pathlib import Path

# SECURITY WARNING: keep the secret key used in production secret!
SECRET_KEY = 'django-insecure-abc123'

DEBUG = True
`}

	if err := se.RemoveCoreSetting("SECRET_KEY"); err != nil {
		t.Fatalf("RemoveCoreSetting: %v", err)
	}

	if strings.Contains(se.Content(), "SECRET_KEY") {
		t.Fatalf("expected SECRET_KEY to be removed, got:\n%s", se.Content())
	}
	if strings.Contains(se.Content(), "SECURITY WARNING") {
		t.Fatalf("expected preceding comment to be removed, got:\n%s", se.Content())
	}
	if !strings.Contains(se.Content(), "DEBUG = True") {
		t.Fatalf("expected DEBUG to remain, got:\n%s", se.Content())
	}
}

func TestSettingsEditorRemoveDatabasesBlock(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: `INSTALLED_APPS = ['django.contrib.admin']

# Database
# https://docs.djangoproject.com/en/5.2/ref/settings/#databases

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}

STATIC_URL = 'static/'
`}

	se.RemoveDatabasesBlock()

	if strings.Contains(se.Content(), "DATABASES") {
		t.Fatalf("expected DATABASES block to be removed, got:\n%s", se.Content())
	}
	if strings.Contains(se.Content(), "sqlite3") {
		t.Fatalf("expected sqlite3 to be removed, got:\n%s", se.Content())
	}
	if !strings.Contains(se.Content(), "STATIC_URL") {
		t.Fatalf("expected STATIC_URL to remain, got:\n%s", se.Content())
	}
}

func TestSettingsEditorSaveWritesToDisk(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "settings.py")
	if err := os.WriteFile(path, []byte("OLD = True\n"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	se, err := NewSettingsEditor(path)
	if err != nil {
		t.Fatalf("NewSettingsEditor failed: %v", err)
	}

	se.SetContent("NEW = True\n")
	if err := se.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "NEW = True\n" {
		t.Fatalf("expected saved content to match, got:\n%s", string(data))
	}
}

func TestSettingsEditorContains(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: "DEBUG = True"}

	if !se.Contains("DEBUG") {
		t.Fatal("expected Contains to find DEBUG")
	}
	if se.Contains("SECRET") {
		t.Fatal("expected Contains not to find SECRET")
	}
}

func TestSettingsEditorConfigureTemplatesDirs(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: `TEMPLATES = [
    {
        'DIRS': [],
    },
]

STATIC_URL = 'static/'`}

	if err := se.ConfigureTemplatesDirs(); err != nil {
		t.Fatalf("ConfigureTemplatesDirs: %v", err)
	}

	if !strings.Contains(se.Content(), "BASE_DIR / 'templates'") {
		t.Fatalf("expected DIRS to be updated, got:\n%s", se.Content())
	}
	if !strings.Contains(se.Content(), "STATICFILES_DIRS") {
		t.Fatalf("expected STATICFILES_DIRS to be added, got:\n%s", se.Content())
	}

	first := se.Content()
	if err := se.ConfigureTemplatesDirs(); err != nil {
		t.Fatalf("second ConfigureTemplatesDirs: %v", err)
	}
	if se.Content() != first {
		t.Fatalf("ConfigureTemplatesDirs should be idempotent")
	}
}

func TestConfigureTemplatesDirsReportsAMissingAnchor(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: `TEMPLATES = [
    {
        "DIRS": [],
    },
]

STATIC_URL = 'static/'`}

	err := se.ConfigureTemplatesDirs()
	if err == nil {
		t.Fatal("ConfigureTemplatesDirs returned nil for settings with no matching anchor")
	}

	var anchorErr *AnchorError
	if !errors.As(err, &anchorErr) {
		t.Fatalf("error is %T, want *AnchorError: %v", err, err)
	}
	if anchorErr.Anchor != "'DIRS': []" {
		t.Errorf("AnchorError.Anchor = %q, want the DIRS anchor", anchorErr.Anchor)
	}
	if !strings.Contains(err.Error(), SettingsFile) {
		t.Errorf("error %q does not name the file it failed on", err)
	}
}

func TestSettingsEditorAddProjectContextProcessor(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: `TEMPLATES = [
    {
        'OPTIONS': {
            'context_processors': [
                'django.contrib.messages.context_processors.messages',
            ],
        },
    },
]`}

	if err := se.AddProjectContextProcessor("myapp"); err != nil {
		t.Fatalf("AddProjectContextProcessor: %v", err)
	}

	if !strings.Contains(se.Content(), "'myapp.context_processors.project_context'") {
		t.Fatalf("expected context processor to be added, got:\n%s", se.Content())
	}

	first := se.Content()
	if err := se.AddProjectContextProcessor("myapp"); err != nil {
		t.Fatalf("second AddProjectContextProcessor: %v", err)
	}
	if se.Content() != first {
		t.Fatalf("AddProjectContextProcessor should be idempotent")
	}
}

func TestAddProjectContextProcessorReportsAMissingAnchor(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: `TEMPLATES = [
    {
        'OPTIONS': {
            'context_processors': [
                'django.template.context_processors.request',
            ],
        },
    },
]`}

	err := se.AddProjectContextProcessor("myapp")
	if err == nil {
		t.Fatal("AddProjectContextProcessor returned nil for settings with no matching anchor")
	}

	var anchorErr *AnchorError
	if !errors.As(err, &anchorErr) {
		t.Fatalf("error is %T, want *AnchorError: %v", err, err)
	}
	if !strings.Contains(anchorErr.Anchor, "messages") {
		t.Errorf("AnchorError.Anchor = %q, want the messages context-processor line", anchorErr.Anchor)
	}
}

func TestConfigureDjangoSettingsFailsOnAMissingAnchor(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "settings.py")

	stub := `from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = 'dev'
DEBUG = True
ALLOWED_HOSTS = []

INSTALLED_APPS = [
    'django.contrib.admin',
]

MIDDLEWARE = [
    'django.middleware.security.SecurityMiddleware',
]

TEMPLATES = [
    {
        'DIRS': [],
        'OPTIONS': {
            'context_processors': [],
        },
    },
]

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}

STATIC_URL = 'static/'
`
	if err := os.WriteFile(path, []byte(stub), 0o644); err != nil {
		t.Fatalf("writing settings stub: %v", err)
	}

	e := &Env{Cfg: config.Project{ProjectName: "demo", DatabaseType: config.SQLite, CreateTemplates: true}}

	err := e.configureSettings(path)
	if err == nil {
		t.Fatal("configureSettings reported success on settings.py it could not fully edit")
	}

	var anchorErr *AnchorError
	if !errors.As(err, &anchorErr) {
		t.Fatalf("error is %T, want *AnchorError: %v", err, err)
	}
}

func TestSettingsEditorInjectEnvConfigPostgres(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: `from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = 'django-insecure-abc'

DEBUG = True

ALLOWED_HOSTS = []

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}
`}

	if err := se.InjectEnvConfig(EnvConfig{Database: "postgres-docker"}); err != nil {
		t.Fatalf("InjectEnvConfig: %v", err)
	}

	if strings.Contains(se.Content(), "sqlite3") {
		t.Fatalf("expected sqlite3 to be removed, got:\n%s", se.Content())
	}
	if !strings.Contains(se.Content(), "django.db.backends.postgresql") {
		t.Fatalf("expected postgresql backend, got:\n%s", se.Content())
	}
	if !strings.Contains(se.Content(), "import environ") {
		t.Fatalf("expected environ import, got:\n%s", se.Content())
	}
	if strings.Count(se.Content(), "DATABASES =") != 1 {
		t.Fatalf("expected exactly 1 DATABASES block, got:\n%s", se.Content())
	}
}

func TestSettingsEditorInjectEnvConfigSQLite(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: `from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = 'django-insecure-abc'

DEBUG = True

ALLOWED_HOSTS = []

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}
`}

	if err := se.InjectEnvConfig(EnvConfig{Database: "sqlite"}); err != nil {
		t.Fatalf("InjectEnvConfig: %v", err)
	}

	if !strings.Contains(se.Content(), "sqlite3") {
		t.Fatalf("expected sqlite3 backend for sqlite type, got:\n%s", se.Content())
	}
	if strings.Contains(se.Content(), "postgresql") {
		t.Fatalf("expected no postgresql for sqlite type, got:\n%s", se.Content())
	}
}

func TestProductionSettingsHaveNoTopLevelConnMaxAge(t *testing.T) {
	t.Parallel()

	for _, mailers := range []bool{false, true} {
		production := render("production.py.tmpl", multiEnvSettingsData{ModuleName: "my_shop", Mailers: mailers, Env: environEnv().productionEnvExpressions()})
		for _, line := range strings.Split(production, "\n") {
			if strings.HasPrefix(line, "CONN_MAX_AGE") {
				t.Errorf("CONN_MAX_AGE is a key of each DATABASES entry; at module level Django ignores it:\n%s", production)
			}
		}
	}
}

func TestHealthChecksAreNotRedirectedToHTTPS(t *testing.T) {
	t.Parallel()

	production := render("production.py.tmpl", multiEnvSettingsData{ModuleName: "my_shop", Env: environEnv().productionEnvExpressions()})
	if !strings.Contains(production, `SECURE_REDIRECT_EXEMPT = [r"^healthz/$"]`) {
		t.Errorf("a load balancer probing /healthz/ over HTTP gets a 301 and marks the app down:\n%s", production)
	}
}

func TestAddProjectContextProcessorIsIdempotent(t *testing.T) {
	t.Parallel()

	settings := `TEMPLATES = [
    {
        'OPTIONS': {
            'context_processors': [
                'django.contrib.messages.context_processors.messages',
            ],
        },
    },
]`

	updated, err := addProjectContextProcessorTo(settings, "demo")
	if err != nil {
		t.Fatalf("first pass: %v", err)
	}
	updatedAgain, err := addProjectContextProcessorTo(updated, "demo")
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}

	if updated != updatedAgain {
		t.Fatalf("expected addProjectContextProcessor to be idempotent")
	}
	if strings.Count(updated, "'demo.context_processors.project_context'") != 1 {
		t.Fatalf("expected context processor to be inserted once, got:\n%s", updated)
	}
}

func TestUpdateSettingsForTemplatesIsIdempotent(t *testing.T) {
	t.Parallel()

	settings := `TEMPLATES = [
    {
        'DIRS': [],
    },
]

STATIC_URL = 'static/'`

	updated, err := configureTemplatesDirsIn(settings)
	if err != nil {
		t.Fatalf("first pass: %v", err)
	}
	updatedAgain, err := configureTemplatesDirsIn(updated)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}

	if updated != updatedAgain {
		t.Fatalf("expected UpdateSettingsForTemplates to be idempotent")
	}
	if strings.Count(updated, "BASE_DIR / 'templates'") != 1 {
		t.Fatalf("expected template DIRS to be injected once, got:\n%s", updated)
	}
	if strings.Count(updated, "STATICFILES_DIRS") != 1 {
		t.Fatalf("expected STATICFILES_DIRS to be injected once, got:\n%s", updated)
	}
}

func TestInjectEnvConfigPostgresDockerRemovesSQLite(t *testing.T) {
	t.Parallel()

	fakeSettings := `"""
Django settings for myproject project.
"""

from pathlib import Path
import os

# Build paths inside the project like this: BASE_DIR / 'subdir'.
BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = 'django-insecure-abc123'

DEBUG = True

ALLOWED_HOSTS = []

INSTALLED_APPS = [
    'django.contrib.admin',
]

# Database
# https://docs.djangoproject.com/en/5.2/ref/settings/#databases

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}

STATIC_URL = 'static/'
`

	env := &Env{Cfg: config.Project{
		ProjectName:  "myproject",
		DatabaseType: config.PostgresDocker,
	}}

	result := injectEnvConfigInto(fakeSettings, env)

	count := strings.Count(result, "DATABASES =")
	if count != 1 {
		t.Fatalf("expected exactly 1 DATABASES = block, got %d.\nResult:\n%s", count, result)
	}

	if !strings.Contains(result, "django.db.backends.postgresql") {
		t.Fatalf("expected postgresql backend in result:\n%s", result)
	}

	if strings.Contains(result, "sqlite3") {
		t.Fatalf("expected no sqlite3 references in result:\n%s", result)
	}

	envPos := strings.Index(result, "env = environ.Env()")
	dbPos := strings.Index(result, "DATABASES =")
	if envPos < 0 || dbPos < 0 || dbPos < envPos {
		t.Fatalf("expected DATABASES to appear after env = environ.Env()\nenv pos=%d, db pos=%d\nResult:\n%s", envPos, dbPos, result)
	}

	duplicateChecks := map[string]int{
		"from pathlib import Path": 1,
		"import os":                0,
		"BASE_DIR =":               1,
		"SECRET_KEY =":             1,
		"DEBUG =":                  1,
		"ALLOWED_HOSTS =":          1,
	}
	for pattern, maxCount := range duplicateChecks {
		count := strings.Count(result, pattern)
		if pattern == "import os" {

			if count > 1 {
				t.Fatalf("expected at most 1 %q, got %d.\nResult:\n%s", pattern, count, result)
			}
		} else if count != maxCount {
			t.Fatalf("expected exactly %d %q, got %d.\nResult:\n%s", maxCount, pattern, count, result)
		}
	}
}

func TestInjectEnvConfigIsIdempotentAcrossDatabaseTypes(t *testing.T) {
	t.Parallel()

	baseSettings := `from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = "django-insecure"
DEBUG = True
ALLOWED_HOSTS = []

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}
`

	tests := []struct {
		name         string
		databaseType config.Database
		engine       string
	}{
		{name: "sqlite", databaseType: config.SQLite, engine: "django.db.backends.sqlite3"},
		{name: "postgres", databaseType: config.PostgresDocker, engine: "django.db.backends.postgresql"},
		{name: "mysql", databaseType: config.MySQLDocker, engine: "django.db.backends.mysql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &Env{Cfg: config.Project{DatabaseType: tt.databaseType}}
			once := injectEnvConfigInto(baseSettings, env)

			if strings.Count(once, "DATABASES =") != 1 {
				t.Fatalf("expected one DATABASES block for %s, got:\n%s", tt.databaseType, once)
			}
			if strings.Contains(once, "django-stack env config") {
				t.Fatalf("the settings carry scaffold markers for %s:\n%s", tt.databaseType, once)
			}
			for _, setting := range []string{"SECRET_KEY = ", "DEBUG = ", "ALLOWED_HOSTS = "} {
				if strings.Count(once, "\n"+setting) != 1 {
					t.Fatalf("expected exactly one %s for %s, got:\n%s", setting, tt.databaseType, once)
				}
			}
			if !strings.Contains(once, tt.engine) {
				t.Fatalf("expected %s settings to include %s, got:\n%s", tt.databaseType, tt.engine, once)
			}
		})
	}
}

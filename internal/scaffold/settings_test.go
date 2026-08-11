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

	se.RemoveCoreSetting("SECRET_KEY")

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

	// Verify idempotency
	first := se.Content()
	if err := se.ConfigureTemplatesDirs(); err != nil {
		t.Fatalf("second ConfigureTemplatesDirs: %v", err)
	}
	if se.Content() != first {
		t.Fatalf("ConfigureTemplatesDirs should be idempotent")
	}
}

// The point of F10: an anchor that no longer matches must be an error, not a
// silent no-op that yields a project with templates never wired up.
func TestConfigureTemplatesDirsReportsAMissingAnchor(t *testing.T) {
	t.Parallel()

	// Django reflowed TEMPLATES and the literal `'DIRS': []` is gone.
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

	// Verify idempotency
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

	// A TEMPLATES block with no messages context processor — the edit has
	// nothing to anchor on, so every template would render with an empty
	// project_name.
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

// An anchor failure must surface through configureSettings rather than
// being swallowed on the way up.
func TestConfigureDjangoSettingsFailsOnAMissingAnchor(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "settings.py")
	// Valid enough to get past the earlier edits, but with no messages
	// context-processor line for AddProjectContextProcessor to key on.
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

	e := &Env{Cfg: config.Project{ProjectName: "demo", DatabaseType: config.SQLite}}

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

	se.InjectEnvConfig("postgres-docker")

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

	se.InjectEnvConfig("sqlite")

	if !strings.Contains(se.Content(), "sqlite3") {
		t.Fatalf("expected sqlite3 backend for sqlite type, got:\n%s", se.Content())
	}
	if strings.Contains(se.Content(), "postgresql") {
		t.Fatalf("expected no postgresql for sqlite type, got:\n%s", se.Content())
	}
}

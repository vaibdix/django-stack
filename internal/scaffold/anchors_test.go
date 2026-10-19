package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

// Every edit below used to fall back when its anchor was missing: append to
// the end, prepend to the file, or do nothing. Each must now say which anchor
// it could not find.
func TestEveryEditRefusesAMissingAnchor(t *testing.T) {
	t.Parallel()

	settings := func(content string) *SettingsEditor { return &SettingsEditor{content: content} }
	cases := map[string]struct {
		edit   func() error
		anchor string
	}{
		"whitenoise without SecurityMiddleware": {
			func() error {
				return settings("from pathlib import Path\n\nMIDDLEWARE = [\n    'a.B',\n]\n").
					AddToListAfter(MiddlewareList, MiddlewareSecurity, MiddlewareWhiteNoise)
			},
			"'" + MiddlewareSecurity + "',",
		},
		"an import into a module with none": {
			func() error { return settings("DEBUG = True\n").AddImport("import structlog") },
			"a top-level import",
		},
		"a setting grouped with one that is absent": {
			func() error {
				return settings("DEBUG = True\n").SetTopLevelAfter("LOGIN_URL", "LOGIN_REDIRECT_URL", "'home'")
			},
			"LOGIN_URL = ",
		},
		"static files without STATIC_URL": {
			func() error {
				return settings("TEMPLATES = [{'DIRS': []}]\n").ConfigureTemplatesDirs()
			},
			staticURLPrefix,
		},
		"STATIC_ROOT without STATIC_URL": {
			func() error {
				return settings("DEBUG = True\n").InsertAfterLine(staticURLPrefix, "STATIC_ROOT = BASE_DIR / 'staticfiles'")
			},
			staticURLPrefix,
		},
	}
	for name, tc := range cases {
		var anchorErr *AnchorError
		if err := tc.edit(); !errors.As(err, &anchorErr) || anchorErr.Anchor != tc.anchor {
			t.Errorf("%s: error = %v, want an AnchorError for %q", name, err, tc.anchor)
		}
	}
}

func TestWhiteNoiseLandsDirectlyAfterSecurityMiddleware(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: "MIDDLEWARE = [\n    'django.middleware.security.SecurityMiddleware',\n    'django.contrib.sessions.middleware.SessionMiddleware',\n]\n"}
	if err := se.AddToListAfter(MiddlewareList, MiddlewareSecurity, MiddlewareWhiteNoise); err != nil {
		t.Fatalf("AddToListAfter: %v", err)
	}
	security := strings.Index(se.content, MiddlewareSecurity)
	whitenoise := strings.Index(se.content, MiddlewareWhiteNoise)
	sessions := strings.Index(se.content, "SessionMiddleware")
	if security >= whitenoise || whitenoise >= sessions {
		t.Errorf("WhiteNoise must sit directly after SecurityMiddleware:\n%s", se.content)
	}
}

func TestImportsKeepTheirSections(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: "from proj.conf import env\n\nDEBUG = True\n", firstParty: "proj"}
	for _, line := range []string{"import structlog", "import os", "from django.utils.csp import CSP"} {
		if err := se.AddImport(line); err != nil {
			t.Fatalf("AddImport(%q): %v", line, err)
		}
	}
	want := "import os\n\nimport structlog\nfrom django.utils.csp import CSP\n\nfrom proj.conf import env\n\nDEBUG = True\n"
	if se.content != want {
		t.Errorf("imports were not grouped stdlib, third party, first party:\ngot:\n%s\nwant:\n%s", se.content, want)
	}
}

func TestPyprojectEditsRefuseWhatTheyCannotPlace(t *testing.T) {
	t.Parallel()

	noFloor := &PyprojectEditor{content: "[project]\nname = \"demo\"\n"}
	var anchorErr *AnchorError
	if err := noFloor.SetProjectDependencies([]string{"django==6.1"}); !errors.As(err, &anchorErr) {
		t.Errorf("dependencies without requires-python: error = %v, want an AnchorError", err)
	}
	if strings.Count(noFloor.content, "[project]") != 1 {
		t.Errorf("a second [project] table was written:\n%s", noFloor.content)
	}

	grouped := &PyprojectEditor{content: "[dependency-groups]\ndev = [\"ruff\"]\n"}
	if err := grouped.SetDependencyGroup("dev", []string{"mypy"}); err == nil {
		t.Error("an existing [dependency-groups] table was kept and the new group silently dropped")
	}

	unreadable := t.TempDir()
	if err := os.Mkdir(filepath.Join(unreadable, PyprojectFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPyprojectEditor(unreadable, "demo", PythonToolingMetadata{RequiresPython: ">=3.12"}); err == nil {
		t.Error("an unreadable pyproject.toml was treated as missing, so Save would overwrite it")
	}
}

func TestMultiEnvRewritesRefuseAMissingAnchor(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: config.Project{ProjectName: "proj"}}
	var anchorErr *AnchorError
	if _, err := e.rebaseBaseDir("DEBUG = True\n"); !errors.As(err, &anchorErr) {
		t.Errorf("base.py without BASE_DIR: error = %v, want an AnchorError", err)
	}

	manage := filepath.Join(t.TempDir(), ManageFile)
	if err := os.WriteFile(manage, []byte("os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'proj.settings.local')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := e.updateSettingsModule(manage, "proj.settings.production"); !errors.As(err, &anchorErr) {
		t.Errorf("a settings module already rewritten: error = %v, want an AnchorError, not proj.settings.production.local", err)
	}

	if err := os.WriteFile(manage, []byte(`os.environ.setdefault("DJANGO_SETTINGS_MODULE", "proj.settings")`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := e.updateSettingsModule(manage, "proj.settings.local"); err != nil {
		t.Fatalf("updateSettingsModule: %v", err)
	}
	if got, _ := os.ReadFile(manage); !strings.Contains(string(got), `"proj.settings.local"`) {
		t.Errorf("the rewrite lost the file's quote style:\n%s", got)
	}
}

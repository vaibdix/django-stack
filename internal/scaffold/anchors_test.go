package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestEveryEditRefusesAMissingAnchor(t *testing.T) {
	t.Parallel()

	settings := func(content string) *settingsEditor { return &settingsEditor{content: content} }
	cases := map[string]struct {
		edit   func() error
		anchor string
	}{
		"whitenoise without SecurityMiddleware": {
			func() error {
				return settings("from pathlib import Path\n\nMIDDLEWARE = [\n    'a.B',\n]\n").
					AddToListAfter(middlewareList, middlewareSecurity, middlewareWhiteNoise)
			},
			"'" + middlewareSecurity + "',",
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
		var anchorErr *anchorError
		if err := tc.edit(); !errors.As(err, &anchorErr) || anchorErr.Anchor != tc.anchor {
			t.Errorf("%s: error = %v, want an AnchorError for %q", name, err, tc.anchor)
		}
	}
}

func TestWhiteNoiseLandsDirectlyAfterSecurityMiddleware(t *testing.T) {
	t.Parallel()

	se := &settingsEditor{content: "MIDDLEWARE = [\n    'django.middleware.security.SecurityMiddleware',\n    'django.contrib.sessions.middleware.SessionMiddleware',\n]\n"}
	if err := se.AddToListAfter(middlewareList, middlewareSecurity, middlewareWhiteNoise); err != nil {
		t.Fatalf("AddToListAfter: %v", err)
	}
	security := strings.Index(se.content, middlewareSecurity)
	whitenoise := strings.Index(se.content, middlewareWhiteNoise)
	sessions := strings.Index(se.content, "SessionMiddleware")
	if security >= whitenoise || whitenoise >= sessions {
		t.Errorf("WhiteNoise must sit directly after SecurityMiddleware:\n%s", se.content)
	}
}

func TestImportsAreAddedOnceAtTheEndOfTheImportBlock(t *testing.T) {
	t.Parallel()

	se := &settingsEditor{content: "from proj.conf import env\n\n\nDEBUG = True\n"}
	for _, line := range []string{"import structlog", "import os", "import structlog"} {
		if err := se.AddImport(line); err != nil {
			t.Fatalf("AddImport(%q): %v", line, err)
		}
	}
	want := "from proj.conf import env\nimport structlog\nimport os\n\nDEBUG = True\n"
	if se.content != want {
		t.Errorf("got:\n%s\nwant:\n%s", se.content, want)
	}
}

func TestPyprojectEditsRefuseWhatTheyCannotPlace(t *testing.T) {
	t.Parallel()

	noFloor := &pyprojectEditor{content: "[project]\nname = \"demo\"\n"}
	var anchorErr *anchorError
	if err := noFloor.SetProjectDependencies([]string{"django==6.1"}); !errors.As(err, &anchorErr) {
		t.Errorf("dependencies without requires-python: error = %v, want an AnchorError", err)
	}
	if strings.Count(noFloor.content, "[project]") != 1 {
		t.Errorf("a second [project] table was written:\n%s", noFloor.content)
	}

	grouped := &pyprojectEditor{content: "[dependency-groups]\ndev = [\"ruff\"]\n"}
	if err := grouped.SetDependencyGroup("dev", []string{"mypy"}); err == nil {
		t.Error("an existing [dependency-groups] table was kept and the new group silently dropped")
	}

	unreadable := t.TempDir()
	if err := os.Mkdir(filepath.Join(unreadable, pyprojectFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := newPyprojectEditor(unreadable, "demo", pythonToolingMetadata{RequiresPython: ">=3.12"}); err == nil {
		t.Error("an unreadable pyproject.toml was treated as missing, so Save would overwrite it")
	}
}

func TestMultiEnvRewritesRefuseAMissingAnchor(t *testing.T) {
	t.Parallel()

	e := &Env{cfg: config.Project{ProjectName: "proj"}}
	var anchorErr *anchorError
	if _, err := e.rebaseBaseDir("DEBUG = True\n"); !errors.As(err, &anchorErr) {
		t.Errorf("base.py without BASE_DIR: error = %v, want an AnchorError", err)
	}

	manage := filepath.Join(t.TempDir(), manageFile)
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

func TestRemovingASettingLeavesSettingsThatOnlyShareItsPrefix(t *testing.T) {
	t.Parallel()

	se := &settingsEditor{content: "DEBUG = True\nDEBUG_PROPAGATE_EXCEPTIONS = False\nSECRET_KEY_FALLBACKS = []\nSECRET_KEY = 'x'\n"}
	for _, name := range []string{"DEBUG", "SECRET_KEY"} {
		if err := se.RemoveCoreSetting(name); err != nil {
			t.Fatalf("RemoveCoreSetting(%s): %v", name, err)
		}
	}
	if se.content != "DEBUG_PROPAGATE_EXCEPTIONS = False\nSECRET_KEY_FALLBACKS = []\n" {
		t.Errorf("removing DEBUG and SECRET_KEY took settings that only share the prefix:\n%s", se.content)
	}

	var anchorErr *anchorError
	if err := se.RemoveCoreSetting("ALLOWED_HOSTS"); !errors.As(err, &anchorErr) {
		t.Errorf("removing a setting that is not there: error = %v, want an AnchorError", err)
	}
}

func TestAMentionOutsideTheListDoesNotCountAsAlreadyAdded(t *testing.T) {
	t.Parallel()

	se := &settingsEditor{content: "INSTALLED_APPS = [\n    'django.contrib.admin',\n]\n\n# see 'corsheaders' docs\nOTHER = ['corsheaders']\n"}
	if err := se.AddToList(installedApps, "corsheaders"); err != nil {
		t.Fatalf("AddToList: %v", err)
	}
	want := "INSTALLED_APPS = [\n    'django.contrib.admin',\n    'corsheaders',\n]\n"
	if !strings.HasPrefix(se.content, want) {
		t.Errorf("the app was skipped because it is mentioned elsewhere:\n%s", se.content)
	}
	if err := se.AddToList(installedApps, "corsheaders"); err != nil || strings.Count(se.content, "    'corsheaders',") != 1 {
		t.Errorf("adding twice was not idempotent (err %v):\n%s", err, se.content)
	}
}

func TestListScanningIgnoresBracketsInStringsAndComments(t *testing.T) {
	t.Parallel()

	se := &settingsEditor{content: "DEBUG_MIDDLEWARE = ['x']\nMIDDLEWARE = [\n    'a.B',  # a ] in a comment\n    'c.D[]',\n]\nAFTER = 1\n"}
	if err := se.AddToList(middlewareList, "e.F"); err != nil {
		t.Fatalf("AddToList: %v", err)
	}
	want := "MIDDLEWARE = [\n    'a.B',  # a ] in a comment\n    'c.D[]',\n    'e.F',\n]\nAFTER = 1\n"
	if !strings.HasSuffix(se.content, want) || !strings.HasPrefix(se.content, "DEBUG_MIDDLEWARE = ['x']\n") {
		t.Errorf("the item went into the wrong list or the wrong place:\n%s", se.content)
	}
}

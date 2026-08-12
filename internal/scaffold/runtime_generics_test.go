package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

// Django's ModelForm and ModelAdmin are generic to a type checker but not at
// runtime, so ModelForm[Book] is a TypeError until django_stubs_ext patches it
// in. Writing the parameter without the patch does not fail a lint — it stops
// the project from importing at all. These two must therefore move together.
func TestParameterisedFormsOnlyAppearWithTheRuntimePatch(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name            string
		runtimeGenerics bool
		wantParameter   bool
	}{
		{name: "mypy on, patch installed", runtimeGenerics: true, wantParameter: true},
		{name: "mypy off, no patch", runtimeGenerics: false, wantParameter: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			forms := generateBookForms(true, tc.runtimeGenerics)
			admin := generateAdminPy(tc.runtimeGenerics)

			if got := strings.Contains(forms, "ModelForm[Book]"); got != tc.wantParameter {
				t.Errorf("ModelForm[Book] present = %v, want %v:\n%s", got, tc.wantParameter, forms)
			}
			if got := strings.Contains(admin, "ModelAdmin[Book]"); got != tc.wantParameter {
				t.Errorf("ModelAdmin[Book] present = %v, want %v:\n%s", got, tc.wantParameter, admin)
			}
		})
	}
}

// DRF's ModelSerializer and ModelViewSet define __class_getitem__ themselves, so
// they are parameterised unconditionally. If that ever stops being true, an
// API-only project without mypy would fail to import.
func TestDRFClassesAreParameterisedWithoutAnyPatch(t *testing.T) {
	t.Parallel()

	if !strings.Contains(generateBookSerializer(), "ModelSerializer[Book]") {
		t.Error("serializers.py lost its type parameter")
	}
	if !strings.Contains(generateBookAPI(false), "ModelViewSet[Book]") {
		t.Error("api.py lost its type parameter")
	}
}

// The patch has to be in settings, which Django imports before any form or
// admin module, and it has to survive the multi-env split into base.py.
func TestRuntimePatchLandsInSettingsAndSurvivesTheSplit(t *testing.T) {
	t.Parallel()

	cfg := config.Project{
		ProjectName:           "demo",
		DjangoVersion:         "6.1",
		SelectedPythonVersion: "Python 3.13.2",
		DatabaseType:          config.SQLite,
		SetupMypy:             true,
	}

	written := mypyConfigFor(t, cfg)
	if !strings.Contains(written, "strict = true") {
		t.Errorf("the generated mypy config is not strict:\n%s", written)
	}
	if !strings.Contains(written, "disallow_any_explicit = true") {
		t.Errorf("the generated mypy config still permits Any:\n%s", written)
	}

	settings := settingsWithRuntimeGenerics(t, cfg)
	if !strings.Contains(settings, "django_stubs_ext.monkeypatch()") {
		t.Fatalf("settings do not install the runtime patch:\n%s", settings)
	}

	e := &Env{Cfg: cfg, Reporter: NopReporter{}}
	if base := e.generateBasePy(settings); !strings.Contains(base, "django_stubs_ext.monkeypatch()") {
		t.Errorf("the multi-env split dropped the runtime patch, so forms would stop importing:\n%s", base)
	}
}

// Both settings layouts need the implicit_reexport relaxation: only the
// multi-env one uses a star import, but the pattern list must cover the single
// module too or a change there goes unnoticed.
func TestMypyOverridesCoverBothSettingsLayouts(t *testing.T) {
	t.Parallel()

	written := mypyConfigFor(t, config.Project{
		ProjectName:           "demo",
		DjangoVersion:         "6.1",
		SelectedPythonVersion: "Python 3.13.2",
	})

	if !strings.Contains(written, `module = ["*.settings", "*.settings.*"]`) {
		t.Errorf("the settings override does not cover both layouts:\n%s", written)
	}
	if !strings.Contains(written, `module = ["*.migrations.*"]`) {
		t.Errorf("generated migrations are not exempt, so Django's own output fails the check:\n%s", written)
	}
}

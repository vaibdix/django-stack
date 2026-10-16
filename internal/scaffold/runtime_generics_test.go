package scaffold

import (
	"regexp"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

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

var djangoClassesNeedingThePatch = regexp.MustCompile(
	`\b(ModelAdmin|UserAdmin|ModelForm|ListView|CreateView|UpdateView|DeleteView|DetailView)\[`)

func TestNothingSubscriptsADjangoClassWithoutTheRuntimePatch(t *testing.T) {
	t.Parallel()

	const app = "catalog"
	sources := map[string]string{
		"accounts/admin.py":    generateAccountsAdmin(false),
		"catalog/admin.py":     generateAdminPy(false),
		"catalog/forms.py":     generateBookForms(true, false),
		"catalog/views.py":     generateBookViews(app, false, false),
		"catalog/views (htmx)": generateBookViews(app, true, false),
	}
	for name, source := range sources {
		if match := djangoClassesNeedingThePatch.FindString(source); match != "" {
			t.Errorf("%s subscripts %s without django_stubs_ext.monkeypatch(), so it raises "+
				"TypeError on import when the Mypy add-on is off:\n%s", name, match, source)
		}
	}
}

func TestTheUserAdminIsParameterisedOnlyWithThePatch(t *testing.T) {
	t.Parallel()

	if !strings.Contains(generateAccountsAdmin(true), "BaseUserAdmin[User]") {
		t.Error("with mypy on, UserAdmin lost its type parameter")
	}
	if strings.Contains(generateAccountsAdmin(false), "BaseUserAdmin[User]") {
		t.Error("with mypy off, UserAdmin is still subscripted")
	}
}

func TestDRFClassesAreParameterisedWithoutAnyPatch(t *testing.T) {
	t.Parallel()

	if !strings.Contains(generateBookSerializer("catalog"), "ModelSerializer[Book]") {
		t.Error("serializers.py lost its type parameter")
	}
	if !strings.Contains(generateBookAPI(), "ModelViewSet[Book]") {
		t.Error("api.py lost its type parameter")
	}
}

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

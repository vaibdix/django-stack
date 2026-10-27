package render

import (
	"regexp"
	"strings"
	"testing"
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

			forms := generateBookForms(bookFormsData{Tailwind: true, RuntimeGenerics: tc.runtimeGenerics})
			admin := generateAdminPy(typingData{RuntimeGenerics: tc.runtimeGenerics})

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
		"accounts/admin.py":    generateAccountsAdmin(typingData{RuntimeGenerics: false}),
		"catalog/admin.py":     generateAdminPy(typingData{RuntimeGenerics: false}),
		"catalog/forms.py":     generateBookForms(bookFormsData{Tailwind: true, RuntimeGenerics: false}),
		"catalog/views.py":     renderBookViews(bookViewsData{AppName: app, HTMX: false, RuntimeGenerics: false}),
		"catalog/views (htmx)": renderBookViews(bookViewsData{AppName: app, HTMX: true, RuntimeGenerics: false}),
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

	if !strings.Contains(generateAccountsAdmin(typingData{RuntimeGenerics: true}), "BaseUserAdmin[User]") {
		t.Error("with mypy on, UserAdmin lost its type parameter")
	}
	if strings.Contains(generateAccountsAdmin(typingData{RuntimeGenerics: false}), "BaseUserAdmin[User]") {
		t.Error("with mypy off, UserAdmin is still subscripted")
	}
}

func TestDRFClassesAreParameterisedWithoutAnyPatch(t *testing.T) {
	t.Parallel()

	if !strings.Contains(generateAPIPackage("catalog")["serializers.py"], "ModelSerializer[Book]") {
		t.Error("serializers.py lost its type parameter")
	}
	if !strings.Contains(generateBookAPI(), "ModelViewSet[Book]") {
		t.Error("api.py lost its type parameter")
	}
}

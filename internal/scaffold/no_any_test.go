package scaffold

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/report"
)

var anyWord = regexp.MustCompile(`\bAny\b`)

func findAnyAnnotation(source string) (string, bool) {
	for _, line := range strings.Split(stripPythonProse(source), "\n") {
		if anyWord.MatchString(line) {
			return strings.TrimSpace(line), true
		}
	}
	return "", false
}

func stripPythonProse(source string) string {
	var kept []string
	inDocstring := false

	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)

		if fences := strings.Count(trimmed, `"""`) + strings.Count(trimmed, "'''"); fences > 0 {
			if fences%2 != 0 {
				inDocstring = !inDocstring
			}
			continue
		}
		if inDocstring || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if idx := strings.Index(line, " #"); idx != -1 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func TestGeneratedPythonDeclaresNoAny(t *testing.T) {
	t.Parallel()

	for name, source := range everyGeneratedPythonSource(t) {
		if form, found := findAnyAnnotation(source); found {
			t.Errorf("%s annotates with %q; object, a TypedDict or a pydantic model says more:\n%s",
				name, form, source)
		}
	}
}

func TestNoPythonTemplateDeclaresAny(t *testing.T) {
	t.Parallel()

	err := fs.WalkDir(templateFS, "templates/python", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		raw, readErr := templateFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if form, found := findAnyAnnotation(string(raw)); found {
			t.Errorf("template %s annotates with %q", filepath.Base(path), form)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the python templates: %v", err)
	}
}

func TestAnyDetectionIgnoresProseButNotCode(t *testing.T) {
	t.Parallel()

	prose := `def f(value: object) -> object:
    """DRF types validated_data as dict[str, Any], which is why this exists."""
    # cleaned_data is dict[str, Any] too
    return value
`
	if form, found := findAnyAnnotation(prose); found {
		t.Errorf("a docstring describing Any was read as an annotation: %q", form)
	}

	for _, code := range []string{
		"def f(value: Any) -> None:\n    return None\n",
		"from typing import cast, Any\nx = cast(Any, y)\n",
		"import typing as t\ndef f(v: t.Any) -> None: ...\n",
	} {
		if _, found := findAnyAnnotation(code); !found {
			t.Errorf("Any was not detected in %q, so this guard proves nothing", code)
		}
	}
}

func everyGeneratedPythonSource(t *testing.T) map[string]string {
	t.Helper()

	const app = "catalog"

	sources := map[string]string{
		"models.py":         generateBookModel("catalog", true),
		"serializers.py":    generateAPIPackage("catalog")["serializers.py"],
		"api.py":            generateBookAPI(),
		"fields.py":         generateBookFields(typingData{RuntimeGenerics: true}),
		"admin.py":          generateAdminPy(typingData{RuntimeGenerics: true}),
		"forms.py":          generateBookForms(bookFormsData{Tailwind: true, RuntimeGenerics: true}),
		"views.py":          renderBookViews(bookViewsData{AppName: app, HTMX: true, RuntimeGenerics: true}),
		"urls.py":           generateAppURLs(app),
		"create_sample.py":  generateSampleData(app, config.SQLite),
		"ninja/api.py":      generateNinjaPackage(app, "my-shop")["api.py"],
		"ninja/router.py":   generateNinjaPackage(app, "my-shop")["router.py"],
		"ninja/schemas.py":  generateNinjaPackage(app, "my-shop")["schemas.py"],
		"ninja/__init__.py": generateNinjaPackage(app, "my-shop")["__init__.py"],
		"conf.py":           render("conf.py.tmpl", pydanticConfData()),
		"settings/local":    render("local.py.tmpl", pydanticSettingsData()),
		"settings/prod":     render("production.py.tmpl", pydanticSettingsData()),
		"settings/test":     render("test.py.tmpl", pydanticSettingsData()),
		"tests.py":          render("tests_book.py.tmpl", testsData{appNames: appNames{AppName: app, AppNameTitle: "Catalog"}}),
		"test_models.py":    render("tests_model.py.tmpl", appNames{AppName: app, AppNameTitle: "Catalog"}),
		"tests.py (ninja)":  render("tests_ninja.py.tmpl", testsData{appNames: appNames{AppName: app, AppNameTitle: "Catalog"}}),
	}

	named := &Env{cfg: config.Project{ProjectName: "demo", AppName: app}, reporter: report.Nop{}}
	anonymous := &Env{cfg: config.Project{ProjectName: "demo"}, reporter: report.Nop{}}
	sources["context_processors.py"] = named.renderContextProcessors()
	sources["context_processors.py (no app)"] = anonymous.renderContextProcessors()

	return sources
}

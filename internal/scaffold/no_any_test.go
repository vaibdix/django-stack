package scaffold

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

var anyAnnotations = []string{": Any", "-> Any", "[Any", "Any]", "import Any", "typing.Any"}

func findAnyAnnotation(source string) (string, bool) {
	code := stripPythonProse(source)
	for _, form := range anyAnnotations {
		if strings.Contains(code, form) {
			return form, true
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

	code := "def f(value: Any) -> None:\n    return None\n"
	if _, found := findAnyAnnotation(code); !found {
		t.Error("a real Any annotation was not detected, so this guard proves nothing")
	}
}

func everyGeneratedPythonSource(t *testing.T) map[string]string {
	t.Helper()

	const app = "catalog"

	sources := map[string]string{
		"models.py":                    generateBookModel("catalog", true, "6.1"),
		"serializers.py":               generateBookSerializer(),
		"api.py":                       generateBookAPI(true),
		"schemas.py":                   generateBookSchemas(),
		"admin.py":                     generateAdminPy(true),
		"forms.py":                     generateBookForms(true, true),
		"views.py":                     generateBookViews(app, true, true),
		"urls.py":                      generateAppURLs(app, true),
		"create_sample.py":             generateSampleData(app, true),
		"create_sample_no_pydantic.py": generateSampleData(app, false),
		"conf.py":                      render("conf.py.tmpl", pydanticConfData()),
		"settings/local":               render("local.py.tmpl", pydanticSettingsData()),
		"settings/prod":                render("production.py.tmpl", pydanticSettingsData()),
		"settings/test":                render("test.py.tmpl", pydanticSettingsData()),
		"tests.py":                     render("tests_book.py.tmpl", appNames{AppName: app, AppNameTitle: "Catalog"}),
	}

	named := &Env{Cfg: config.Project{ProjectName: "demo", AppName: app}, Reporter: NopReporter{}}
	anonymous := &Env{Cfg: config.Project{ProjectName: "demo"}, Reporter: NopReporter{}}
	sources["context_processors.py"] = named.renderContextProcessors()
	sources["context_processors.py (no app)"] = anonymous.renderContextProcessors()

	return sources
}

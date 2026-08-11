package scaffold

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// render panics on failure rather than returning an error, on the grounds that
// a broken embedded template is a build defect and not something a caller can
// handle. That reasoning only holds if a broken template actually fails a
// build — which is what this test is for.

// everyTemplateField carries every field any embedded template references.
// Templates needing only a subset ignore the rest; a template referencing a
// field that is not here fails TestEveryTemplateParsesAndRenders, which is the
// signal to add it.
type everyTemplateField struct {
	AppName      string
	AppNameTitle string
	HTMX         bool
	Tailwind     bool
	InputClass   string
	ProjectPath  string
	VSCodePath   string
	RuffEnabled  bool
	DjangoTrack  string
}

func fullTemplateData(htmx bool) everyTemplateField {
	return everyTemplateField{
		AppName:      "catalog",
		AppNameTitle: "Catalog",
		HTMX:         htmx,
		Tailwind:     true,
		InputClass:   "cls",
		ProjectPath:  "/home/dev/my-shop",
		VSCodePath:   "/usr/local/bin/code",
		RuffEnabled:  true,
		DjangoTrack:  "5.2",
	}
}

// TestEveryTemplateParsesAndRenders executes every embedded template against a
// fully populated data value, in both HTMX modes. A template referencing a
// field that does not exist, or using an action the parser rejects, fails here
// instead of panicking inside a user's scaffold.
func TestEveryTemplateParsesAndRenders(t *testing.T) {
	t.Parallel()

	names, err := templateNames()
	if err != nil {
		t.Fatalf("walking embedded templates: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no embedded templates found; the //go:embed pattern is not matching")
	}

	for _, name := range names {
		for _, htmx := range []bool{false, true} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("template %s panicked with HTMX=%v: %v", name, htmx, r)
					}
				}()

				got := render(name, fullTemplateData(htmx))
				if strings.TrimSpace(got) == "" {
					t.Errorf("template %s rendered empty output", name)
				}
			})
		}
	}
}

// No rendered template may still contain an unexpanded action: that means a
// delimiter was mistyped and the literal would be written into the project.
func TestRenderedTemplatesLeaveNoUnexpandedActions(t *testing.T) {
	t.Parallel()

	names, err := templateNames()
	if err != nil {
		t.Fatalf("walking embedded templates: %v", err)
	}

	for _, name := range names {
		got := render(name, fullTemplateData(true))

		if strings.Contains(got, templateLeftDelim) {
			t.Errorf("template %s rendered output still containing %q — a mistyped action", name, templateLeftDelim)
		}
	}
}

// templateNames lists the base names of every embedded template, which is the
// key ExecuteTemplate expects.
func templateNames() ([]string, error) {
	var names []string
	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			names = append(names, filepath.Base(path))
		}
		return nil
	})
	return names, err
}

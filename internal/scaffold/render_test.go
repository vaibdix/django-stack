package scaffold

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

type everyTemplateField struct {
	AppName         string
	AppNameTitle    string
	HTMX            bool
	Tailwind        bool
	InputClass      string
	ProjectPath     string
	VSCodePath      string
	RuffEnabled     bool
	DjangoTrack     string
	StartScriptName string
	SuperuserGuide  string
	ProjectDirName  string
}

func fullTemplateData(htmx bool) everyTemplateField {
	return everyTemplateField{
		AppName:         "catalog",
		AppNameTitle:    "Catalog",
		HTMX:            htmx,
		Tailwind:        true,
		InputClass:      "cls",
		ProjectPath:     "/home/dev/my-shop",
		VSCodePath:      "/usr/local/bin/code",
		RuffEnabled:     true,
		DjangoTrack:     "5.2",
		StartScriptName: "start-dev.sh",
		SuperuserGuide:  "SUPERUSER_SETUP.md",
		ProjectDirName:  "my-shop",
	}
}

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

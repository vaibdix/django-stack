package scaffold

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

type everyTemplateField struct {
	AppName           string
	AppNameTitle      string
	HTMX              bool
	Tailwind          bool
	InputClass        string
	RuffEnabled       bool
	MypyEnabled       bool
	UVAvailable       bool
	DjangoTrack       string
	DjLintVersion     string
	MypyEntry         string
	ProjectDirName    string
	AllAuth           bool
	Mailers           bool
	Database          bool
	DBUser            string
	DBHost            string
	UpsertTarget      bool
	DBPort            string
	Sentry            bool
	Structlog         bool
	ModuleName        string
	PythonVersion     string
	DatabaseLabel     string
	TestCommand       string
	AdminURL          string
	Docker            bool
	MultiEnv          bool
	REST              bool
	Ruff              bool
	Mypy              bool
	PreCommit         bool
	Templates         bool
	UV                bool
	Python            string
	Tool              string
	ProjectName       string
	Pydantic          bool
	RuntimeGenerics   bool
	HTMLViews         bool
	Postgres          bool
	ConstraintKeyword string
	Env               productionEnvExpressions
}

func fullTemplateData(htmx bool) everyTemplateField {
	return everyTemplateField{
		AppName:           "catalog",
		AppNameTitle:      "Catalog",
		HTMX:              htmx,
		Tailwind:          true,
		InputClass:        "cls",
		RuffEnabled:       true,
		MypyEnabled:       true,
		UVAvailable:       true,
		DjangoTrack:       "5.2",
		ProjectDirName:    "my-shop",
		AllAuth:           true,
		Mailers:           true,
		Database:          true,
		DBUser:            "postgres",
		DBHost:            "localhost",
		UpsertTarget:      true,
		DBPort:            "5432",
		Sentry:            true,
		Structlog:         true,
		ModuleName:        "my_shop",
		PythonVersion:     "3.13",
		DatabaseLabel:     "PostgreSQL",
		TestCommand:       "python manage.py test",
		AdminURL:          "admin/",
		Docker:            true,
		MultiEnv:          true,
		REST:              true,
		Ruff:              true,
		Mypy:              true,
		PreCommit:         true,
		Templates:         true,
		ProjectName:       "my-shop",
		Pydantic:          true,
		RuntimeGenerics:   true,
		HTMLViews:         true,
		Postgres:          true,
		ConstraintKeyword: "condition",
		Env:               (&Env{}).productionEnvExpressions(),
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

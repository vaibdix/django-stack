package scaffold

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/golden"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/dbdocker"
)

type countingReader struct{ next byte }

func (r *countingReader) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = r.next
		r.next++
	}
	return len(b), nil
}

func TestGoldenAPIGenerators(t *testing.T) {
	t.Parallel()

	const app = "catalog"

	cases := map[string]string{
		"api/app_urls.py":            generateAppURLs(app),
		"api/book_serializer.py":     generateAPIPackage("catalog")["serializers.py"],
		"api/book_forms_plain.py":    generateBookForms(bookFormsData{Tailwind: false, RuntimeGenerics: true}),
		"api/book_forms_tailwind.py": generateBookForms(bookFormsData{Tailwind: true, RuntimeGenerics: true}),
		"api/book_api.py":            generateBookAPI(),
		"api/book_model.py":          generateBookModel("catalog", true, config.SQLite),
		"api/book_views_plain.py":    renderBookViews(bookViewsData{AppName: app, HTMX: false, RuntimeGenerics: true}),
		"api/book_views_htmx.py":     renderBookViews(bookViewsData{AppName: app, HTMX: true, RuntimeGenerics: true}),
		"api/book_views_stubs52.py":  renderBookViews(bookViewsData{AppName: app, HTMX: true, RuntimeGenerics: true, Stubs52: true}),
		"api/admin.py":               generateAdminPy(typingData{RuntimeGenerics: true}),
		"api/sample_data.py":         generateSampleData(app, config.SQLite),
	}

	for name, got := range cases {
		t.Run(name, func(t *testing.T) {
			golden.Assert(t, name, got)
		})
	}
}

func TestGoldenEnvFiles(t *testing.T) {
	t.Parallel()

	databases := []config.Database{config.SQLite, config.PostgresDocker, config.MySQLDocker}

	for _, db := range databases {
		t.Run(string(db), func(t *testing.T) {
			t.Parallel()

			e := &Env{Cfg: withCredentials(config.Project{
				ProjectName:  "my-shop",
				DatabaseType: db,
			}), entropy: &countingReader{}}

			golden.Assert(t, "env/"+string(db)+".env", envFileFor(t, e))
			golden.Assert(t, "env/"+string(db)+".env.example", e.renderEnvExampleContent())
		})
	}
}

func TestGoldenEnvFilesReuseDockerCredentials(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: withCredentials(config.Project{
		ProjectName:            "my-shop",
		DatabaseType:           config.PostgresDocker,
		PostgresDockerPort:     "55432",
		PostgresDockerPassword: "fixedpassword123",
	}), entropy: &countingReader{}}

	golden.Assert(t, "env/postgres-docker-preset.env", envFileFor(t, e))
}

func TestEnvAndExampleDeclareTheSameVariables(t *testing.T) {
	databases := []config.Database{config.SQLite, config.PostgresDocker, config.MySQLDocker}

	for _, db := range databases {
		t.Run(string(db), func(t *testing.T) {
			e := &Env{Cfg: withCredentials(config.Project{ProjectName: "my-shop", DatabaseType: db})}

			envKeys := dotenvKeys(envFileFor(t, e))
			exampleKeys := dotenvKeys(e.renderEnvExampleContent())

			for _, key := range envKeys {
				if !slices.Contains(exampleKeys, key) {
					t.Errorf(".env sets %s but .env.example never mentions it", key)
				}
			}
		})
	}
}

func TestEveryEnvExampleKeyIsReadByTheProject(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{config.SQLite, config.PostgresDocker, config.MySQLDocker} {
		for _, pydantic := range []bool{false, true} {
			cfg := withCredentials(config.Project{ProjectName: "my-shop", DatabaseType: db, SetupPydanticConf: pydantic})
			e := &Env{Cfg: cfg}
			path := newSettingsFixture(t)
			if err := e.configureSettings(path); err != nil {
				t.Fatalf("configureSettings: %v", err)
			}
			readers := readFile(t, filepath.Dir(path), filepath.Base(path))
			if pydantic {
				readers += render("conf.py.tmpl", e.confTemplateData())
			}
			if db.UsesDocker() {
				dir := t.TempDir()
				if err := dbdocker.WriteComposeFile(dir, cfg.ProjectName, db); err != nil {
					t.Fatalf("WriteComposeFile: %v", err)
				}
				readers += readFile(t, dir, dbdocker.ComposeFile)
			}
			for _, key := range dotenvKeys(e.renderEnvExampleContent()) {
				if !strings.Contains(readers, key) && !strings.Contains(readers, strings.ToLower(key)) {
					t.Errorf("%s pydantic=%t: .env.example sets %s, which nothing reads", db, pydantic, key)
				}
			}
		}
	}
}

func dotenvKeys(content string) []string {
	var keys []string
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if name, _, ok := strings.Cut(line, "="); ok {
			keys = append(keys, name)
		}
	}
	return keys
}

func appTemplateVariants(appName string) map[string]AppTemplateConfig {
	return map[string]AppTemplateConfig{
		"plain":         {AppName: appName},
		"htmx":          {AppName: appName, HTMX: true},
		"tailwind":      {AppName: appName, Tailwind: true},
		"tailwind_htmx": {AppName: appName, Tailwind: true, HTMX: true},
	}
}

func TestGoldenAppTemplates(t *testing.T) {
	t.Parallel()

	pages := []string{"book_list", "book_form", "book_delete"}

	for variant, cfg := range appTemplateVariants("catalog") {
		for _, page := range pages {
			t.Run(variant+"/"+page, func(t *testing.T) {
				t.Parallel()
				golden.Assert(t, "app_templates/"+variant+"/"+page+".html", cfg.render(page))
			})
		}
	}
}

func TestGoldenPartials(t *testing.T) {
	t.Parallel()

	for variant, cfg := range appTemplateVariants("catalog") {
		pages := []string{"book_list_items", "book_fields"}
		if cfg.HTMX {
			pages = append(pages, "book_form_partial")
		}
		for _, page := range pages {
			t.Run(variant+"/"+page, func(t *testing.T) {
				t.Parallel()
				golden.Assert(t, "app_templates/"+variant+"/partials/"+page+".html", cfg.render(page))
			})
		}
	}
}

func TestGoldenMultiEnvSettings(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: config.Project{
		ProjectName:  "my-shop",
		DatabaseType: config.PostgresDocker,
		SetupRuff:    true,
	}}

	const originalSettings = `from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = env("SECRET_KEY")
DEBUG = env.bool("DEBUG", default=False)
ALLOWED_HOSTS = env.list("ALLOWED_HOSTS", default=[])

INSTALLED_APPS = [
    "django.contrib.admin",
]
`

	environExpressions := environEnv().productionEnvExpressions()
	mailers := multiEnvSettingsData{ModuleName: "my_shop", Templates: true, AllAuth: true, Mailers: true, NativeCSP: true, Env: environExpressions, Ruff: true}
	legacy := multiEnvSettingsData{ModuleName: "my_shop", Templates: true, AllAuth: true, Mailers: false, NativeCSP: true, Env: environExpressions, Ruff: true}
	typed := multiEnvSettingsData{ModuleName: "my_shop", Templates: true, AllAuth: true, Mailers: true, NativeCSP: true, Env: pydanticEnv().productionEnvExpressions(), Ruff: true}

	cases := map[string]string{
		"multienv/base.py": basePyFor(t, e, originalSettings),

		"multienv/local.py":      render("local.py.tmpl", mailers),
		"multienv/production.py": render("production.py.tmpl", mailers),
		"multienv/test.py":       render("test.py.tmpl", mailers),

		"multienv/local_email_backend.py":      render("local.py.tmpl", legacy),
		"multienv/production_email_backend.py": render("production.py.tmpl", legacy),
		"multienv/test_email_backend.py":       render("test.py.tmpl", legacy),

		"multienv/production_typed_settings.py": render("production.py.tmpl", typed),
	}

	for name, got := range cases {
		t.Run(name, func(t *testing.T) {
			golden.Assert(t, name, got)
		})
	}
}

func TestGoldenStartScripts(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"scripts/start-dev-unix-plain.sh":    generateStartScript(false, false),
		"scripts/start-dev-unix-tailwind.sh": generateStartScript(false, true),
		"scripts/start-dev-windows.bat":      generateStartScript(true, false),
	}

	for name, got := range cases {
		t.Run(name, func(t *testing.T) {
			golden.Assert(t, name, got)
		})
	}
}

func TestGoldenAdminPy(t *testing.T) {
	t.Parallel()

	golden.Assert(t, "app/admin.py", generateAdminPy(typingData{RuntimeGenerics: false}))
}

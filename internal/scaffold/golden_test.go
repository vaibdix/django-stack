package scaffold

import (
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/golden"

	"github.com/vaibdix/django-cli/internal/config"
)

func withDeterministicSecrets(t *testing.T) {
	t.Helper()

	original := randRead
	var counter byte
	randRead = func(b []byte) {
		for i := range b {
			b[i] = counter
			counter++
		}
	}
	t.Cleanup(func() { randRead = original })
}

func TestGoldenAPIGenerators(t *testing.T) {
	t.Parallel()

	const app = "catalog"

	cases := map[string]string{
		"api/app_urls.py":            generateAppURLs(app),
		"api/book_serializer.py":     generateBookSerializer("catalog"),
		"api/book_forms_plain.py":    generateBookForms(false, true),
		"api/book_forms_tailwind.py": generateBookForms(true, true),
		"api/book_api.py":            generateBookAPI(),
		"api/book_model.py":          generateBookModel("catalog", true, "6.1", config.SQLite),
		"api/book_views_plain.py":    generateBookViews(app, false, true),
		"api/book_views_htmx.py":     generateBookViews(app, true, true),
		"api/admin.py":               generateAdminPy(true),
		"api/sample_data.py":         generateSampleData(app, config.SQLite),
	}

	for name, got := range cases {
		t.Run(name, func(t *testing.T) {
			golden.Assert(t, name, got)
		})
	}
}

func TestGoldenEnvFiles(t *testing.T) {
	databases := []config.Database{config.SQLite, config.PostgresDocker, config.MySQLDocker}

	for _, db := range databases {
		t.Run(string(db), func(t *testing.T) {
			withDeterministicSecrets(t)

			e := &Env{Cfg: withCredentials(config.Project{
				ProjectName:  "my-shop",
				DatabaseType: db,
			})}

			golden.Assert(t, "env/"+string(db)+".env", envFileFor(t, e))
			golden.Assert(t, "env/"+string(db)+".env.example", e.renderEnvExampleContent())
		})
	}
}

func TestGoldenEnvFilesReuseDockerCredentials(t *testing.T) {
	withDeterministicSecrets(t)

	e := &Env{Cfg: withCredentials(config.Project{
		ProjectName:            "my-shop",
		DatabaseType:           config.PostgresDocker,
		PostgresDockerPort:     "55432",
		PostgresDockerPassword: "fixedpassword123",
	})}

	golden.Assert(t, "env/postgres-docker-preset.env", envFileFor(t, e))
}

func TestEnvAndExampleDeclareTheSameVariables(t *testing.T) {
	databases := []config.Database{config.SQLite, config.PostgresDocker, config.MySQLDocker}

	for _, db := range databases {
		t.Run(string(db), func(t *testing.T) {
			withDeterministicSecrets(t)

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

	renderers := map[string]func(AppTemplateConfig) string{
		"book_list.html":   generateBookListTemplate,
		"book_form.html":   generateBookFormTemplate,
		"book_delete.html": generateBookDeleteTemplate,
	}

	for variant, cfg := range appTemplateVariants("catalog") {
		for name, render := range renderers {
			t.Run(variant+"/"+name, func(t *testing.T) {
				t.Parallel()
				golden.Assert(t, "app_templates/"+variant+"/"+name, render(cfg))
			})
		}
	}
}

func TestGoldenHTMXPartials(t *testing.T) {
	t.Parallel()

	renderers := map[string]func(AppTemplateConfig) string{
		"book_list_items.html":   generateBookListItemsPartial,
		"book_form_partial.html": generateBookFormPartial,
	}

	for _, variant := range []string{"htmx", "tailwind_htmx"} {
		cfg := appTemplateVariants("catalog")[variant]
		for name, render := range renderers {
			t.Run(variant+"/"+name, func(t *testing.T) {
				t.Parallel()
				golden.Assert(t, "app_templates/"+variant+"/partials/"+name, render(cfg))
			})
		}
	}
}

func TestGoldenMultiEnvSettings(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: config.Project{
		ProjectName:  "my-shop",
		DatabaseType: config.PostgresDocker,
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
	mailers := multiEnvSettingsData{ModuleName: "my_shop", Templates: true, AllAuth: true, Mailers: true, Env: environExpressions}
	legacy := multiEnvSettingsData{ModuleName: "my_shop", Templates: true, AllAuth: true, Mailers: false, Env: environExpressions}
	typed := multiEnvSettingsData{ModuleName: "my_shop", Templates: true, AllAuth: true, Mailers: true, Env: pydanticEnv().productionEnvExpressions()}

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

	golden.Assert(t, "app/admin.py", generateAdminPy(false))
}

package scaffold

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files instead of comparing against them")

const goldenDir = "testdata/golden"

func assertGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join(goldenDir, name)

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating golden dir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("writing golden %s: %v", name, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden %s: %v\n\nRun: make goldens", name, err)
	}

	if got != string(want) {
		t.Errorf("generated output for %s does not match the golden file.\n"+
			"If this change is intentional, run:\n"+
			"  make goldens\n"+
			"and review the diff.\n\n%s", name, firstDifference(string(want), got))
	}
}

func firstDifference(want, got string) string {
	wantLines := splitLines(want)
	gotLines := splitLines(got)

	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			return "first difference at line " + itoa(i+1) + ":\n" +
				"  want: " + quote(w) + "\n" +
				"  got:  " + quote(g)
		}
	}
	return "files differ only in trailing content"
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func quote(s string) string {
	const limit = 100
	if len(s) > limit {
		s = s[:limit] + "..."
	}
	return "\"" + s + "\""
}

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
		"api/book_api.py":            generateBookAPI(false),
		"api/book_model.py":          generateBookModel("catalog", true, "6.1", config.SQLite),
		"api/book_views_plain.py":    generateBookViews(app, false, false, true),
		"api/book_views_htmx.py":     generateBookViews(app, true, false, true),
		"api/admin.py":               generateAdminPy(true),
		"api/sample_data.py":         generateSampleData(app, false),
	}

	for name, got := range cases {
		t.Run(name, func(t *testing.T) {
			assertGolden(t, name, got)
		})
	}
}

func TestGoldenEnvFiles(t *testing.T) {
	databases := []config.Database{config.SQLite, config.PostgresDocker, config.PostgresString, config.MySQLDocker}

	for _, db := range databases {
		t.Run(string(db), func(t *testing.T) {
			withDeterministicSecrets(t)

			e := &Env{Cfg: config.Project{
				ProjectName:  "my-shop",
				DatabaseType: db,
			}}

			assertGolden(t, "env/"+string(db)+".env", e.renderEnvFileContent())
			assertGolden(t, "env/"+string(db)+".env.example", e.renderEnvExampleContent())
		})
	}
}

func TestGoldenEnvFilesReuseDockerCredentials(t *testing.T) {
	withDeterministicSecrets(t)

	e := &Env{Cfg: config.Project{
		ProjectName:            "my-shop",
		DatabaseType:           config.PostgresDocker,
		PostgresDockerPort:     "55432",
		PostgresDockerPassword: "fixedpassword123",
	}}

	assertGolden(t, "env/postgres-docker-preset.env", e.renderEnvFileContent())
}

func TestEnvAndExampleDeclareTheSameVariables(t *testing.T) {
	databases := []config.Database{config.SQLite, config.PostgresDocker, config.PostgresString, config.MySQLDocker}

	for _, db := range databases {
		t.Run(string(db), func(t *testing.T) {
			withDeterministicSecrets(t)

			e := &Env{Cfg: config.Project{ProjectName: "my-shop", DatabaseType: db}}

			envKeys := dotenvKeys(e.renderEnvFileContent())
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
				assertGolden(t, "app_templates/"+variant+"/"+name, render(cfg))
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
				assertGolden(t, "app_templates/"+variant+"/partials/"+name, render(cfg))
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
	mailers := multiEnvSettingsData{AllAuth: true, Mailers: true, Env: environExpressions}
	legacy := multiEnvSettingsData{AllAuth: true, Mailers: false, Env: environExpressions}
	typed := multiEnvSettingsData{AllAuth: true, Mailers: true, Env: pydanticEnv().productionEnvExpressions()}

	cases := map[string]string{
		"multienv/base.py": e.generateBasePy(originalSettings),

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
			assertGolden(t, name, got)
		})
	}
}

func TestGoldenStartScripts(t *testing.T) {
	t.Parallel()

	const (
		projectPath = "/home/dev/my-shop"
		vscodePath  = "/usr/local/bin/code"
	)

	unix := func(tailwind bool) string {
		return render("start-dev.sh.tmpl", startScriptData{VSCodePath: vscodePath, Tailwind: tailwind})
	}

	cases := map[string]string{
		"scripts/start-dev-unix-plain.sh":    unix(false),
		"scripts/start-dev-unix-tailwind.sh": unix(true),
		"scripts/start-dev-windows.bat": render("start-dev.bat.tmpl",
			startScriptData{ProjectPath: projectPath, VSCodePath: vscodePath}),
	}

	for name, got := range cases {
		t.Run(name, func(t *testing.T) {
			assertGolden(t, name, got)
		})
	}
}

func TestGoldenAdminPy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		rest bool
	}{
		{name: "app/admin_plain.py", rest: false},
		{name: "app/admin_rest.py", rest: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &Env{Cfg: config.Project{
				AppName:            "catalog",
				SetupRestFramework: tc.rest,
			}}
			assertGolden(t, tc.name, e.adminPyContent())
		})
	}
}

func TestGoldenAPIEndpointsMarkdown(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: config.Project{
		ProjectName:        "my-shop",
		AppName:            "catalog",
		SetupRestFramework: true,
	}}

	assertGolden(t, "api/API_ENDPOINTS.md", e.generateAPIEndpointsMd())
}

func TestGoldenSuperuserGuide(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := writeSuperuserSetupGuide(dir); err != nil {
		t.Fatalf("writing superuser guide: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading temp dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one generated file, got %d", len(entries))
	}

	got, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("reading guide: %v", err)
	}
	assertGolden(t, "onboarding/"+entries[0].Name(), string(got))
}

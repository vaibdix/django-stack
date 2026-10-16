package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"errors"
	"runtime"

	"github.com/vaibdix/django-cli/internal/assets"
	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func TestRenderEnvFileContentSQLiteKeepsDefaultDBName(t *testing.T) {
	t.Parallel()

	env := &Env{
		Cfg: config.Project{
			ProjectName:  "demo-project",
			DatabaseType: config.SQLite,
		},
	}

	content := env.renderEnvFileContent()

	if strings.Contains(content, "DB_ENGINE") || strings.Contains(content, "DB_NAME") {
		t.Fatalf("sqlite env content should not have DB_ENGINE or DB_NAME (uses hardcoded settings), got:\n%s", content)
	}
	if strings.Contains(content, "DATABASE_URL=postgres") {
		t.Fatalf("sqlite env content should not have postgres DATABASE_URL, got:\n%s", content)
	}
}

func TestRenderEnvFileContentPostgresIsExplicit(t *testing.T) {
	t.Parallel()

	env := &Env{
		Cfg: config.Project{
			ProjectName:            "demo-project",
			DatabaseType:           config.PostgresDocker,
			PostgresDockerPort:     "5544",
			PostgresDockerPassword: "secretpass",
		},
	}

	content := env.renderEnvFileContent()

	checks := map[string]bool{
		"DB_NAME=demo_project":   strings.Contains(content, "DB_NAME=demo_project"),
		"DB_USER=postgres":       strings.Contains(content, "DB_USER=postgres"),
		"DB_PASSWORD=secretpass": strings.Contains(content, "DB_PASSWORD=secretpass"),
		"DB_PORT=5544":           strings.Contains(content, "DB_PORT=5544"),
	}
	for key, found := range checks {
		if !found {
			t.Fatalf("expected postgres env content to contain %s, got:\n%s", key, content)
		}
	}
}

func TestRenderEnvExampleContentPostgresString(t *testing.T) {
	t.Parallel()

	env := &Env{
		Cfg: config.Project{
			ProjectName:  "sample-api",
			DatabaseType: config.PostgresString,
		},
	}

	content := env.renderEnvExampleContent()

	if !strings.Contains(content, "DB_NAME=sample_api") {
		t.Fatalf("expected postgres example to include DB_NAME=sample_api, got:\n%s", content)
	}
	if !strings.Contains(content, "DB_PASSWORD=") {
		t.Fatalf("expected postgres example to include DB_PASSWORD=, got:\n%s", content)
	}
}

func TestEveryAppRegistersTheBookAdmin(t *testing.T) {
	t.Parallel()

	content := generateAdminPy(false)
	for _, fragment := range []string{
		"from .models import Book",
		"@admin.register(Book)",
		"list_display = (\"title\", \"author\", \"isbn\", \"publication_date\", \"price\", \"created_at\")",
		"readonly_fields = (\"created_at\", \"updated_at\")",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("admin.py is missing %q:\n%s", fragment, content)
		}
	}
}

func TestTheReadmeDocumentsEverythingAnOperatorNeeds(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	e := &Env{
		Cfg: config.Project{
			ProjectName: "demo", AppName: "catalog", DjangoVersion: "6.1",
			SelectedPythonVersion: "Python 3.13.2", DatabaseType: config.PostgresDocker,
			SetupRuff: true, SetupMypy: true, SetupMultiEnv: true, SetupRestFramework: true,
		},
		Reporter: NopReporter{},
	}

	if err := e.writeReadme(projectPath); err != nil {
		t.Fatalf("writeReadme: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(projectPath, "README.md"))
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	readme := string(content)

	for _, fragment := range []string{
		"uv sync",
		"docker compose up -d --wait db",
		"cp .env.example .env",
		"manage.py migrate",
		"manage.py createsuperuser",
		"./.venv/bin/python manage.py createsuperuser",
		`.venv\Scripts\python.exe manage.py createsuperuser`,
		"python manage.py test --settings=demo.settings.test",
		"ruff check .",
		"mypy .",
		"collectstatic",
		"gunicorn demo.wsgi:application",
		"/healthz/",
		"ENVIRONMENT",
	} {
		if !strings.Contains(readme, fragment) {
			t.Errorf("README.md never mentions %q, so that step is documented nowhere:\n%s", fragment, readme)
		}
	}
}

func TestTailwindTemplateContentMatchesTheReferenceLayout(t *testing.T) {
	t.Parallel()

	base := assets.RenderPage(assets.PageBase, assets.PageOptions{
		Tailwind: true,
		AppName:  "library",
		API:      true,
	})
	if !strings.Contains(base, "{% url 'swagger-ui' %}") {
		t.Fatalf("expected tailwind base template to include API docs nav link, got:\n%s", base)
	}
	if !strings.Contains(base, `class="frame"`) {
		t.Fatalf("expected tailwind base template to wrap the page in the framed column, got:\n%s", base)
	}
	if !strings.Contains(base, `id="next-steps"`) {
		t.Fatalf("expected the reference's closing band, got:\n%s", base)
	}
	if strings.Contains(base, `id="spacer"`) {
		t.Fatalf("the reference's spacer closes its column, but this layout has a footer doing that; "+
			"keeping both is 88px of dead height, got:\n%s", base)
	}
	if strings.Contains(base, "white/") {
		t.Fatalf("expected the base template to style with tokens rather than opacity literals, got:\n%s", base)
	}
	if strings.Contains(base, `class="dark"`) {
		t.Fatalf("the layout pins a colour scheme; the reference follows prefers-color-scheme, got:\n%s", base)
	}

	index := assets.RenderPage(assets.PageIndex, assets.PageOptions{Tailwind: true})
	if !strings.Contains(index, "Get started") {
		t.Fatalf("expected tailwind index template to include the reference heading, got:\n%s", index)
	}
	if !strings.Contains(index, "Open admin") {
		t.Fatalf("expected tailwind index template to include admin CTA, got:\n%s", index)
	}
	if !strings.Contains(index, "button button-primary") || !strings.Contains(index, "button button-secondary") {
		t.Fatalf("expected the index CTAs to use the shared button classes, got:\n%s", index)
	}

}

func TestCreateExampleAPIWritesRefinedBookTemplates(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	appPath := filepath.Join(projectPath, "library")
	if err := os.MkdirAll(appPath, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	apiEnv := &Env{Cfg: config.Project{AppName: "library", SetupTailwind: true, CreateTemplates: true}}
	if err := apiEnv.writeExampleApp(projectPath, generateAPIPackage(apiEnv.Cfg.AppName, apiEnv.Cfg.SetupPydanticConf)); err != nil {
		t.Fatalf("writeExampleApp failed: %v", err)
	}

	bookListBytes, err := os.ReadFile(filepath.Join(appPath, "templates", "library", "book_list.html"))
	if err != nil {
		t.Fatalf("failed to read generated book_list.html: %v", err)
	}
	bookList := string(bookListBytes)
	if !strings.Contains(bookList, "Book library") {
		t.Fatalf("expected the book list heading, got:\n%s", bookList)
	}
	if !strings.Contains(bookList, "Open admin") {
		t.Fatalf("expected book list to include admin CTA, got:\n%s", bookList)
	}

	bookFormBytes, err := os.ReadFile(filepath.Join(appPath, "templates", "library", "book_form.html"))
	if err != nil {
		t.Fatalf("failed to read generated book_form.html: %v", err)
	}
	if !strings.Contains(string(bookFormBytes), "{{ action }} book") {
		t.Fatalf("expected the form heading to carry the action, got:\n%s", string(bookFormBytes))
	}

	bookDeleteBytes, err := os.ReadFile(filepath.Join(appPath, "templates", "library", "book_delete.html"))
	if err != nil {
		t.Fatalf("failed to read generated book_delete.html: %v", err)
	}
	if !strings.Contains(string(bookDeleteBytes), "Delete permanently") {
		t.Fatalf("expected refined delete CTA, got:\n%s", string(bookDeleteBytes))
	}

	formsBytes, err := os.ReadFile(filepath.Join(appPath, "forms.py"))
	if err != nil {
		t.Fatalf("failed to read generated forms.py: %v", err)
	}
	if !strings.Contains(string(formsBytes), "border border-rule bg-ground") {
		t.Fatalf("expected token-based input classes in forms.py, got:\n%s", string(formsBytes))
	}
}

func TestPackageInstallCommandUsesVenvPythonWithoutUV(t *testing.T) {
	t.Setenv("PATH", "")

	projectPath := filepath.Join(t.TempDir(), "demo")
	command, args := pyenv.PackageInstallCommand(projectPath)

	expectedPython := filepath.Join(projectPath, ".venv")
	if !strings.Contains(command, expectedPython) {
		t.Fatalf("expected fallback command to target .venv under %q, got %q", projectPath, command)
	}
	if len(args) < 3 || args[0] != "-m" || args[1] != "pip" || args[2] != "install" {
		t.Fatalf("expected pip install args, got %#v", args)
	}
}

func TestInstallPackagesUsesVenvTargetWithUV(t *testing.T) {
	t.Parallel()

	if !hasUV() {
		t.Skip("uv is not installed on this machine")
	}

	projectPath := filepath.Join(t.TempDir(), "demo")
	path, args := pyenv.PackageInstallArgs(projectPath, "django==6.0.3")
	expectedPython := filepath.Join(projectPath, ".venv")

	if filepath.Base(path) != "uv" && args[0] != "uv" {
		t.Fatalf("expected uv command, got path=%q args=%#v", path, args)
	}
	if !strings.Contains(strings.Join(args, " "), expectedPython) {
		t.Fatalf("expected uv pip to target venv python under %q, got %q", projectPath, strings.Join(args, " "))
	}
	if strings.Contains(strings.Join(args, " "), "--system") {
		t.Fatalf("install command must not target system Python: %q", strings.Join(args, " "))
	}
}

func TestDevServerCommandUsesVenvInterpreterOnly(t *testing.T) {
	t.Parallel()

	command := devServerCommandFor(runtime.GOOS == "windows")
	if runtime.GOOS == "windows" {
		if command != `.venv\Scripts\python.exe manage.py runserver` {
			t.Fatalf("unexpected Windows dev server command: %q", command)
		}
		return
	}
	if command != "./.venv/bin/python manage.py runserver" {
		t.Fatalf("unexpected Unix dev server command: %q", command)
	}
	if strings.Contains(command, "uv run") {
		t.Fatalf("dev server command should not depend on uv: %q", command)
	}
}

func TestAddProjectContextProcessorIsIdempotent(t *testing.T) {
	t.Parallel()

	settings := `TEMPLATES = [
    {
        'OPTIONS': {
            'context_processors': [
                'django.contrib.messages.context_processors.messages',
            ],
        },
    },
]`

	updated, err := addProjectContextProcessorTo(settings, "demo")
	if err != nil {
		t.Fatalf("first pass: %v", err)
	}
	updatedAgain, err := addProjectContextProcessorTo(updated, "demo")
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}

	if updated != updatedAgain {
		t.Fatalf("expected addProjectContextProcessor to be idempotent")
	}
	if strings.Count(updated, "'demo.context_processors.project_context'") != 1 {
		t.Fatalf("expected context processor to be inserted once, got:\n%s", updated)
	}
}

func TestUpdateSettingsForTemplatesIsIdempotent(t *testing.T) {
	t.Parallel()

	settings := `TEMPLATES = [
    {
        'DIRS': [],
    },
]

STATIC_URL = 'static/'`

	updated, err := configureTemplatesDirsIn(settings)
	if err != nil {
		t.Fatalf("first pass: %v", err)
	}
	updatedAgain, err := configureTemplatesDirsIn(updated)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}

	if updated != updatedAgain {
		t.Fatalf("expected UpdateSettingsForTemplates to be idempotent")
	}
	if strings.Count(updated, "BASE_DIR / 'templates'") != 1 {
		t.Fatalf("expected template DIRS to be injected once, got:\n%s", updated)
	}
	if strings.Count(updated, "STATICFILES_DIRS") != 1 {
		t.Fatalf("expected STATICFILES_DIRS to be injected once, got:\n%s", updated)
	}
}

func TestBuildProjectURLsContentMatchesEnabledFeatures(t *testing.T) {
	t.Parallel()

	minimal := (&Env{}).buildProjectURLsContent()
	if strings.Contains(minimal, "HomeView") || strings.Contains(minimal, "SpectacularAPIView") {
		t.Fatalf("minimal URL config should not reference template views, got:\n%s", minimal)
	}

	apiOnly := (&Env{Cfg: config.Project{
		SetupRestFramework: true,
		AppName:            "library",
	}}).buildProjectURLsContent()
	if !strings.Contains(apiOnly, "api-auth/") || !strings.Contains(apiOnly, "api/v1/") {
		t.Fatalf("API URL config should include REST routes, got:\n%s", apiOnly)
	}
	if !strings.Contains(apiOnly, "SpectacularAPIView") {
		t.Fatalf("API URL config should include drf-spectacular routes, got:\n%s", apiOnly)
	}

	full := (&Env{Cfg: config.Project{
		CreateTemplates:    true,
		CreateAppTemplates: true,
		SetupRestFramework: true,
		AppName:            "library",
	}}).buildProjectURLsContent()
	required := []string{"views.HomeView", "SpectacularSwaggerView", "SpectacularRedocView", "api-auth/", "api/v1/", "library/"}
	for _, fragment := range required {
		if !strings.Contains(full, fragment) {
			t.Fatalf("full URL config missing %q, got:\n%s", fragment, full)
		}
	}
}

func TestInjectEnvConfigPostgresDockerRemovesSQLite(t *testing.T) {
	t.Parallel()

	fakeSettings := `"""
Django settings for myproject project.
"""

from pathlib import Path
import os

# Build paths inside the project like this: BASE_DIR / 'subdir'.
BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = 'django-insecure-abc123'

DEBUG = True

ALLOWED_HOSTS = []

INSTALLED_APPS = [
    'django.contrib.admin',
]

# Database
# https://docs.djangoproject.com/en/5.2/ref/settings/#databases

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}

STATIC_URL = 'static/'
`

	env := &Env{Cfg: config.Project{
		ProjectName:  "myproject",
		DatabaseType: config.PostgresDocker,
	}}

	result := injectEnvConfigInto(fakeSettings, env)

	count := strings.Count(result, "DATABASES =")
	if count != 1 {
		t.Fatalf("expected exactly 1 DATABASES = block, got %d.\nResult:\n%s", count, result)
	}

	if !strings.Contains(result, "django.db.backends.postgresql") {
		t.Fatalf("expected postgresql backend in result:\n%s", result)
	}

	if strings.Contains(result, "sqlite3") {
		t.Fatalf("expected no sqlite3 references in result:\n%s", result)
	}

	envPos := strings.Index(result, "env = environ.Env()")
	dbPos := strings.Index(result, "DATABASES =")
	if envPos < 0 || dbPos < 0 || dbPos < envPos {
		t.Fatalf("expected DATABASES to appear after env = environ.Env()\nenv pos=%d, db pos=%d\nResult:\n%s", envPos, dbPos, result)
	}

	duplicateChecks := map[string]int{
		"from pathlib import Path": 1,
		"import os":                0,
		"BASE_DIR =":               1,
		"SECRET_KEY =":             1,
		"DEBUG =":                  1,
		"ALLOWED_HOSTS =":          1,
	}
	for pattern, maxCount := range duplicateChecks {
		count := strings.Count(result, pattern)
		if pattern == "import os" {

			if count > 1 {
				t.Fatalf("expected at most 1 %q, got %d.\nResult:\n%s", pattern, count, result)
			}
		} else if count != maxCount {
			t.Fatalf("expected exactly %d %q, got %d.\nResult:\n%s", maxCount, pattern, count, result)
		}
	}
}

func TestInjectEnvConfigIsIdempotentAcrossDatabaseTypes(t *testing.T) {
	t.Parallel()

	baseSettings := `from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = "django-insecure"
DEBUG = True
ALLOWED_HOSTS = []

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': BASE_DIR / 'db.sqlite3',
    }
}
`

	tests := []struct {
		name         string
		databaseType config.Database
		engine       string
	}{
		{name: "sqlite", databaseType: config.SQLite, engine: "django.db.backends.sqlite3"},
		{name: "postgres", databaseType: config.PostgresDocker, engine: "django.db.backends.postgresql"},
		{name: "mysql", databaseType: config.MySQLDocker, engine: "django.db.backends.mysql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &Env{Cfg: config.Project{DatabaseType: tt.databaseType}}
			once := injectEnvConfigInto(baseSettings, env)
			twice := injectEnvConfigInto(once, env)

			if once != twice {
				t.Fatalf("expected InjectEnvConfig to be idempotent for %s.\nOnce:\n%s\n\nTwice:\n%s", tt.databaseType, once, twice)
			}
			if strings.Count(once, "DATABASES =") != 1 {
				t.Fatalf("expected one DATABASES block for %s, got:\n%s", tt.databaseType, once)
			}
			if strings.Count(once, "# === django-stack env config start ===") != 1 || strings.Count(once, "# === django-stack env config end ===") != 1 {
				t.Fatalf("expected exactly one managed env config block for %s, got:\n%s", tt.databaseType, once)
			}
			if !strings.Contains(once, tt.engine) {
				t.Fatalf("expected %s settings to include %s, got:\n%s", tt.databaseType, tt.engine, once)
			}
		})
	}
}

func TestGenerateStartScriptOpensVSCodeWithoutStartingDuplicateServers(t *testing.T) {
	t.Parallel()

	script := generateStartScript("/tmp/demo", "/usr/bin/code", true)
	if !strings.Contains(script, "code") || !strings.Contains(script, "Workspace: Start Dev") {
		t.Fatalf("expected script to explain VS Code workspace task startup, got:\n%s", script)
	}
	if strings.Contains(script, "nohup ./.venv/bin/python manage.py runserver") || strings.Contains(script, "nohup npm run watch:css") {
		t.Fatalf("script should not start duplicate background processes anymore, got:\n%s", script)
	}
}

func TestBuildScaffoldFormattingPlanIncludesMandatoryPackagesAndTemplateFiles(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectPath, "templates"), 0755); err != nil {
		t.Fatalf("failed to create global templates dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectPath, "library", "templates", "library"), 0755); err != nil {
		t.Fatalf("failed to create app templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "templates", "base.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("failed to write global template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "library", "templates", "library", "book_list.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("failed to write app template: %v", err)
	}

	plan, err := buildScaffoldFormattingPlan(projectPath)
	if err != nil {
		t.Fatalf("buildScaffoldFormattingPlan failed: %v", err)
	}
	packages, commands, templateFiles := plan.Packages, plan.Commands, plan.TemplateFiles

	if len(packages) != 2 || packages[0] != "ruff" || packages[1] != "djlint" {
		t.Fatalf("expected mandatory formatter packages [ruff djlint], got %#v", packages)
	}
	if len(commands) != 2 {
		t.Fatalf("expected two mandatory Python formatting commands, got %#v", commands)
	}
	if commands[0].Module != "ruff" || !strings.HasPrefix(strings.Join(commands[0].Args, " "), "check --fix --select ") {
		t.Fatalf("unexpected first formatting command: %#v", commands[0])
	}
	if !commands[0].Advisory {
		t.Fatalf("the autofix pass must be advisory; a lint opinion aborting a scaffold is the bug " +
			"that broke every project which did not enable the Ruff feature")
	}
	if commands[1].Advisory {
		t.Fatalf("ruff format must stay fatal: it is what catches generated code that does not parse")
	}
	if commands[1].Module != "ruff" || strings.Join(commands[1].Args, " ") != "format ." {
		t.Fatalf("unexpected second formatting command: %#v", commands[1])
	}
	if len(templateFiles) != 2 {
		t.Fatalf("expected two template files for djlint, got %#v", templateFiles)
	}
}

func TestBuildScaffoldFormattingPlanSkipsTemplatesWhenNoneExist(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()

	plan, err := buildScaffoldFormattingPlan(projectPath)
	if err != nil {
		t.Fatalf("buildScaffoldFormattingPlan failed: %v", err)
	}
	templateFiles := plan.TemplateFiles
	if len(templateFiles) != 0 {
		t.Fatalf("expected no template files when project has none, got %#v", templateFiles)
	}
}

func TestRunMandatoryScaffoldFormattingUsesVenvCommandsWithoutOptionalFlags(t *testing.T) {
	projectPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectPath, "templates"), 0755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "templates", "base.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	env := &Env{
		Cfg: config.Project{
			SetupRuff:      false,
			SetupPreCommit: false,
		},
	}

	var installedPackages []string
	restoreInstall := withFormattingInstallHook(func(_ *Env, _ string, packages ...string) error {
		installedPackages = append([]string{}, packages...)
		return nil
	})
	defer restoreInstall()

	var called []string
	var sawContext bool
	restoreRunner := withPythonModuleRunner(func(ctx context.Context, _ *Env, projectPath, module string, args ...string) ([]byte, error) {

		sawContext = ctx != nil && ctx.Err() == nil
		called = append(called, formatCommandKey(projectPath, module, args...))
		return []byte("ok"), nil
	})
	defer restoreRunner()

	if err := env.formatGeneratedCode(projectPath); err != nil {
		t.Fatalf("formatGeneratedCode failed: %v", err)
	}

	if strings.Join(installedPackages, ",") != "ruff,djlint" {
		t.Fatalf("expected mandatory formatter install regardless of options, got %#v", installedPackages)
	}
	if len(called) != 3 {
		t.Fatalf("expected ruff check, ruff format, and djlint commands, got %#v", called)
	}
	if !strings.Contains(called[0], filepath.Join(projectPath, ".venv")) || !strings.Contains(called[1], filepath.Join(projectPath, ".venv")) || !strings.Contains(called[2], filepath.Join(projectPath, ".venv")) {
		t.Fatalf("expected formatting commands to target the project venv, got %#v", called)
	}
	if !strings.Contains(called[2], "djlint --reformat --profile=django") {
		t.Fatalf("expected direct djlint template formatting command, got %#v", called)
	}
	if !sawContext {
		t.Error("the formatting commands ran without a live context; quitting mid-format would not stop ruff or djlint")
	}
}

func TestRunMandatoryScaffoldFormattingWarnsButContinuesOnRuffAutofixError(t *testing.T) {
	projectPath := t.TempDir()
	rec := &RecordingReporter{}
	env := &Env{Reporter: rec}

	restoreInstall := withFormattingInstallHook(func(_ *Env, _ string, _ ...string) error {
		return nil
	})
	defer restoreInstall()

	restoreRunner := withPythonModuleRunner(func(_ context.Context, _ *Env, _ string, module string, args ...string) ([]byte, error) {
		if module == "ruff" && len(args) >= 2 && args[0] == "check" && args[1] == "--fix" {
			return []byte("E501 line too long"), errors.New("exit status 1")
		}
		return []byte("ok"), nil
	})
	defer restoreRunner()

	err := env.formatGeneratedCode(projectPath)
	if err != nil {
		t.Fatalf("a lint finding must not abort the scaffold, got %v", err)
	}

	steps := strings.Join(rec.Steps, "\n")
	if !strings.Contains(steps, "ruff autofix reported issues") {
		t.Fatalf("the unfixable finding was swallowed silently; steps were:\n%s", steps)
	}
	if !strings.Contains(steps, "Generated code and templates formatted.") {
		t.Fatalf("formatting should still complete; steps were:\n%s", steps)
	}
}

func TestRunMandatoryScaffoldFormattingFailsOnTemplateFormattingError(t *testing.T) {
	projectPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectPath, "templates"), 0755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "templates", "base.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	env := &Env{}

	restoreInstall := withFormattingInstallHook(func(_ *Env, _ string, _ ...string) error {
		return nil
	})
	defer restoreInstall()

	restoreRunner := withPythonModuleRunner(func(_ context.Context, _ *Env, _ string, module string, _ ...string) ([]byte, error) {
		if module == "djlint" {
			return []byte("template parse error"), errors.New("exit status 1")
		}
		return []byte("ok"), nil
	})
	defer restoreRunner()

	err := env.formatGeneratedCode(projectPath)
	if err == nil {
		t.Fatalf("expected formatting failure when djlint fails")
	}
	if !strings.Contains(err.Error(), "template formatting") {
		t.Fatalf("expected error to mention template formatting stage, got %v", err)
	}
}

func TestGeneratedBookViewsUseGetObjectOr404ForStaleIDs(t *testing.T) {
	t.Parallel()

	for _, htmx := range []bool{false, true} {
		content := generateBookViews("library", htmx, false, true)
		if strings.Contains(content, "Book.objects.get(") {
			t.Fatalf("a stale id would raise DoesNotExist rather than 404 when htmx=%v, got:\n%s", htmx, content)
		}
		for _, want := range []string{"UpdateView", "DeleteView"} {
			if !strings.Contains(content, want) {
				t.Fatalf("edit and delete should inherit the 404 that %s gives them when htmx=%v, got:\n%s", want, htmx, content)
			}
		}
	}
}

func TestPythonToolingMetadataMatchesSelectedInterpreter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		option          pyenv.Option
		wantRequires    string
		wantRuffTarget  string
		wantMypyVersion string
	}{
		{
			name:            "python_38",
			option:          pyenv.Option{Path: "/tmp/python38", Version: "3.8.18", Major: 3, Minor: 8, Patch: 18},
			wantRequires:    ">=3.8",
			wantRuffTarget:  "py38",
			wantMypyVersion: "3.8",
		},
		{
			name:            "python_312",
			option:          pyenv.Option{Path: "/tmp/python312", Version: "3.12.4", Major: 3, Minor: 12, Patch: 4},
			wantRequires:    ">=3.12",
			wantRuffTarget:  "py312",
			wantMypyVersion: "3.12",
		},
		{
			name:            "python_314",
			option:          pyenv.Option{Path: "/tmp/python314", Version: "3.14.3", Major: 3, Minor: 14, Patch: 3},
			wantRequires:    ">=3.14",
			wantRuffTarget:  "py314",
			wantMypyVersion: "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &Env{
				Cfg: config.Project{
					SelectedPythonPath:    tt.option.Path,
					SelectedPythonVersion: tt.option.Version,
				},
			}

			requires, ruffTarget, mypyVersion := pythonToolingMetadataOf(env)
			if requires != tt.wantRequires || ruffTarget != tt.wantRuffTarget || mypyVersion != tt.wantMypyVersion {
				t.Fatalf("unexpected tooling metadata for %s: got (%s, %s, %s), want (%s, %s, %s)", tt.name, requires, ruffTarget, mypyVersion, tt.wantRequires, tt.wantRuffTarget, tt.wantMypyVersion)
			}
		})
	}
}

func hasUV() bool {
	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, "uv")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func formatCommandKey(projectPath, module string, args ...string) string {
	return filepath.Join(projectPath, ".venv") + " :: " + module + " " + strings.Join(args, " ")
}

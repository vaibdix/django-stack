package ui_test

import (
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	projecttemplates "django-stack/internal/templates"
	ui "django-stack/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func TestParsePythonVersionOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		major  int
		minor  int
		ok     bool
	}{
		{name: "python_38", output: "Python 3.8.18", major: 3, minor: 8, ok: true},
		{name: "python_311", output: "Python 3.11.2", major: 3, minor: 11, ok: true},
		{name: "python_312", output: "Python 3.12.2", major: 3, minor: 12, ok: true},
		{name: "fallback_fields", output: "3 11\n", major: 3, minor: 11, ok: true},
		{name: "invalid", output: "unknown", major: 0, minor: 0, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, ok := ui.ParsePythonVersionOutputForTest(tt.output)
			if major != tt.major || minor != tt.minor || ok != tt.ok {
				t.Fatalf("ParsePythonVersionOutputForTest(%q) = (%d, %d, %v), want (%d, %d, %v)", tt.output, major, minor, ok, tt.major, tt.minor, tt.ok)
			}
		})
	}
}

func TestBuildDjangoOptionsFromReleaseIndexFiltersAndSortsByCompatibility(t *testing.T) {
	releases := map[string][]ui.TestDjangoReleaseFile{
		"6.0.3":  {{UploadTimeISO8601: "2026-02-03T10:00:00Z"}},
		"5.2.12": {{UploadTimeISO8601: "2026-02-03T09:00:00Z"}},
		"5.1.6":  {{UploadTimeISO8601: "2026-01-15T09:00:00Z"}},
		"4.2.28": {{UploadTimeISO8601: "2025-12-01T09:00:00Z"}},
		"6.0rc1": {{UploadTimeISO8601: "2026-01-01T09:00:00Z"}},
	}

	py314 := ui.PythonOption{Major: 3, Minor: 14, Version: "3.14.3"}
	options314 := ui.BuildDjangoOptionsFromReleaseIndexForTest(py314, releases)
	if len(options314) != 2 {
		t.Fatalf("expected 2 compatible releases for Python 3.14, got %d", len(options314))
	}
	if options314[0].Version != "6.0.3" || options314[1].Version != "5.2.12" {
		t.Fatalf("unexpected release ordering for Python 3.14: %#v", options314)
	}

	py311 := ui.PythonOption{Major: 3, Minor: 11, Version: "3.11.14"}
	options311 := ui.BuildDjangoOptionsFromReleaseIndexForTest(py311, releases)
	expected := []string{"5.2.12", "5.1.6", "4.2.28"}
	if len(options311) != len(expected) {
		t.Fatalf("expected %d compatible releases for Python 3.11, got %d", len(expected), len(options311))
	}
	for i, version := range expected {
		if options311[i].Version != version {
			t.Fatalf("expected options311[%d] = %s, got %s", i, version, options311[i].Version)
		}
	}
}

func TestFallbackDjangoOptionsForPython314ReturnsLatestCompatibleStableReleases(t *testing.T) {
	py314 := ui.PythonOption{Major: 3, Minor: 14, Version: "3.14.3"}
	options := ui.FallbackDjangoOptionsForPythonForTest(py314)
	expected := []string{"6.0.3", "5.2.12", "6.0.2", "5.2.11", "6.0.1"}

	if len(options) != len(expected) {
		t.Fatalf("expected %d fallback options, got %d", len(expected), len(options))
	}
	for i, version := range expected {
		if options[i].Version != version {
			t.Fatalf("expected fallback[%d] = %s, got %s", i, version, options[i].Version)
		}
	}
}

func TestFetchDjangoReleaseOptionsFallsBackWhenPyPIIsUnavailable(t *testing.T) {
	restoreClient := ui.SetDjangoReleaseHTTPClientForTest(&http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network unavailable")
		}),
	})
	defer restoreClient()

	options, note := ui.FetchDjangoReleaseOptionsFromPyPIForTest(ui.PythonOption{Major: 3, Minor: 14, Version: "3.14.3"})
	if len(options) == 0 {
		t.Fatalf("expected bundled fallback options when PyPI is unavailable")
	}
	if !strings.HasPrefix(note, "Showing bundled") {
		t.Fatalf("expected bundled fallback note, got %q", note)
	}
}

func TestRenderEnvFileContentSQLiteKeepsDefaultDBName(t *testing.T) {
	model := &ui.Model{
		ProjectConfig: ui.ProjectConfig{
			ProjectName:  "demo-project",
			DatabaseType: "sqlite",
		},
	}

	content := ui.RenderEnvFileContentForTest(model)

	if strings.Contains(content, "DB_ENGINE") || strings.Contains(content, "DB_NAME") {
		t.Fatalf("sqlite env content should not have DB_ENGINE or DB_NAME (uses hardcoded settings), got:\n%s", content)
	}
	if strings.Contains(content, "DATABASE_URL=postgres") {
		t.Fatalf("sqlite env content should not have postgres DATABASE_URL, got:\n%s", content)
	}
}

func TestRenderEnvFileContentPostgresIsExplicit(t *testing.T) {
	model := &ui.Model{
		ProjectConfig: ui.ProjectConfig{
			ProjectName:            "demo-project",
			DatabaseType:           "postgres-docker",
			PostgresDockerPort:     "5544",
			PostgresDockerPassword: "secretpass",
		},
	}

	content := ui.RenderEnvFileContentForTest(model)

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
	model := &ui.Model{
		ProjectConfig: ui.ProjectConfig{
			ProjectName:  "sample-api",
			DatabaseType: "postgres-string",
		},
	}

	content := ui.RenderEnvExampleContentForTest(model)

	if !strings.Contains(content, "DB_NAME=sample_api") {
		t.Fatalf("expected postgres example to include DB_NAME=sample_api, got:\n%s", content)
	}
	if !strings.Contains(content, "DB_PASSWORD=") {
		t.Fatalf("expected postgres example to include DB_PASSWORD=, got:\n%s", content)
	}
}

func TestAdminPyContentUsesBookAdminForExampleRESTApps(t *testing.T) {
	restContent := ui.AdminPyContentForTest(&ui.Model{
		ProjectConfig: ui.ProjectConfig{
			AppName:            "library",
			SetupRestFramework: true,
		},
	})
	required := []string{
		"from .models import Book",
		"@admin.register(Book)",
		"list_display = (\"title\", \"author\", \"isbn\", \"publication_date\", \"price\", \"created_at\")",
		"readonly_fields = (\"created_at\", \"updated_at\")",
	}
	for _, fragment := range required {
		if !strings.Contains(restContent, fragment) {
			t.Fatalf("expected REST admin content to contain %q, got:\n%s", fragment, restContent)
		}
	}

	placeholderContent := ui.AdminPyContentForTest(&ui.Model{ProjectConfig: ui.ProjectConfig{AppName: "library"}})
	if !strings.Contains(placeholderContent, "# Register your models here.") {
		t.Fatalf("expected non-REST admin content to keep placeholder, got:\n%s", placeholderContent)
	}
	if strings.Contains(placeholderContent, "BookAdmin") {
		t.Fatalf("non-REST placeholder admin should not register BookAdmin, got:\n%s", placeholderContent)
	}
}

func TestWriteSuperuserSetupGuideUsesVenvCommands(t *testing.T) {
	projectPath := t.TempDir()

	if err := ui.WriteSuperuserSetupGuideForTest(projectPath); err != nil {
		t.Fatalf("WriteSuperuserSetupGuideForTest failed: %v", err)
	}

	contentBytes, err := os.ReadFile(filepath.Join(projectPath, "SUPERUSER_SETUP.md"))
	if err != nil {
		t.Fatalf("failed to read SUPERUSER_SETUP.md: %v", err)
	}
	content := string(contentBytes)

	required := []string{
		"./.venv/bin/python manage.py createsuperuser",
		`.venv\Scripts\python.exe manage.py createsuperuser`,
		"Do not use plain `python manage.py ...` unless you have activated the project venv first.",
		"`/admin/`",
	}
	for _, fragment := range required {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected SUPERUSER_SETUP.md to contain %q, got:\n%s", fragment, content)
		}
	}
}

func TestTailwindTemplateContentUsesRefreshedNextInspiredCopy(t *testing.T) {
	base := projecttemplates.TemplateContent("base.html", map[string]string{
		"tailwind": "true",
		"app_name": "library",
		"api":      "true",
	})
	if !strings.Contains(base, "Next.js-inspired Django scaffold") {
		t.Fatalf("expected tailwind base template to include refreshed scaffold copy, got:\n%s", base)
	}
	if !strings.Contains(base, "/api/schema/swagger-ui/") {
		t.Fatalf("expected tailwind base template to include API docs nav link, got:\n%s", base)
	}
	if strings.Contains(base, "from-white/10") {
		t.Fatalf("expected bright white hero wash to be removed from base template, got:\n%s", base)
	}

	index := projecttemplates.TemplateContent("index.html", map[string]string{
		"tailwind": "true",
	})
	if !strings.Contains(index, "Ship Django products with a cleaner first impression.") {
		t.Fatalf("expected tailwind index template to include refreshed hero, got:\n%s", index)
	}
	if !strings.Contains(index, "Launch the admin") {
		t.Fatalf("expected tailwind index template to include admin CTA, got:\n%s", index)
	}

	// api-docs.html template is no longer generated (replaced by drf-spectacular swagger-ui)
}

func TestCreateExampleAPIWritesRefinedBookTemplates(t *testing.T) {
	projectPath := t.TempDir()
	appPath := filepath.Join(projectPath, "library")
	if err := os.MkdirAll(appPath, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	if err := ui.CreateExampleAPIForTest(&ui.Model{ProjectConfig: ui.ProjectConfig{AppName: "library", SetupTailwind: true}}, projectPath); err != nil {
		t.Fatalf("CreateExampleAPIForTest failed: %v", err)
	}

	bookListBytes, err := os.ReadFile(filepath.Join(appPath, "templates", "library", "book_list.html"))
	if err != nil {
		t.Fatalf("failed to read generated book_list.html: %v", err)
	}
	bookList := string(bookListBytes)
	if !strings.Contains(bookList, "Book inventory with a calmer, sharper control surface.") {
		t.Fatalf("expected refined book list hero copy, got:\n%s", bookList)
	}
	if !strings.Contains(bookList, "Open Django admin") {
		t.Fatalf("expected book list to include admin CTA, got:\n%s", bookList)
	}

	bookFormBytes, err := os.ReadFile(filepath.Join(appPath, "templates", "library", "book_form.html"))
	if err != nil {
		t.Fatalf("failed to read generated book_form.html: %v", err)
	}
	if !strings.Contains(string(bookFormBytes), "Structured editor") {
		t.Fatalf("expected refined form layout marker, got:\n%s", string(bookFormBytes))
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
	if !strings.Contains(string(formsBytes), "w-full rounded-2xl border border-white/10") {
		t.Fatalf("expected refined input classes in forms.py, got:\n%s", string(formsBytes))
	}
}

func TestPackageInstallCommandUsesVenvPythonWithoutUV(t *testing.T) {
	t.Setenv("PATH", "")

	projectPath := filepath.Join(t.TempDir(), "demo")
	command, args := ui.PackageInstallCommandForTest(projectPath)

	expectedPython := filepath.Join(projectPath, ".venv")
	if !strings.Contains(command, expectedPython) {
		t.Fatalf("expected fallback command to target .venv under %q, got %q", projectPath, command)
	}
	if len(args) < 3 || args[0] != "-m" || args[1] != "pip" || args[2] != "install" {
		t.Fatalf("expected pip install args, got %#v", args)
	}
}

func TestInstallPackagesUsesVenvTargetWithUV(t *testing.T) {
	if !hasUV() {
		t.Skip("uv is not installed on this machine")
	}

	projectPath := filepath.Join(t.TempDir(), "demo")
	path, args := ui.InstallPackagesCommandForTest(projectPath, "django==6.0.3")
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
	command := ui.DevServerCommandForTest()
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
	settings := `TEMPLATES = [
    {
        'OPTIONS': {
            'context_processors': [
                'django.contrib.messages.context_processors.messages',
            ],
        },
    },
]`

	updated := ui.AddProjectContextProcessorForTest(settings, "demo")
	updatedAgain := ui.AddProjectContextProcessorForTest(updated, "demo")

	if updated != updatedAgain {
		t.Fatalf("expected addProjectContextProcessor to be idempotent")
	}
	if strings.Count(updated, "'demo.context_processors.project_context'") != 1 {
		t.Fatalf("expected context processor to be inserted once, got:\n%s", updated)
	}
}

func TestUpdateSettingsForTemplatesIsIdempotent(t *testing.T) {
	settings := `TEMPLATES = [
    {
        'DIRS': [],
    },
]

STATIC_URL = 'static/'`

	updated := ui.UpdateSettingsForTemplatesForTest(settings)
	updatedAgain := ui.UpdateSettingsForTemplatesForTest(updated)

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
	minimal := ui.BuildProjectURLsContentForTest(&ui.Model{})
	if strings.Contains(minimal, "HomeView") || strings.Contains(minimal, "SpectacularAPIView") {
		t.Fatalf("minimal URL config should not reference template views, got:\n%s", minimal)
	}

	apiOnly := ui.BuildProjectURLsContentForTest(&ui.Model{ProjectConfig: ui.ProjectConfig{
		SetupRestFramework: true,
		AppName:            "library",
	}})
	if !strings.Contains(apiOnly, "api-auth/") || !strings.Contains(apiOnly, "api/v1/") {
		t.Fatalf("API URL config should include REST routes, got:\n%s", apiOnly)
	}
	if !strings.Contains(apiOnly, "SpectacularAPIView") {
		t.Fatalf("API URL config should include drf-spectacular routes, got:\n%s", apiOnly)
	}

	full := ui.BuildProjectURLsContentForTest(&ui.Model{ProjectConfig: ui.ProjectConfig{
		CreateTemplates:    true,
		CreateAppTemplates: true,
		SetupRestFramework: true,
		AppName:            "library",
	}})
	required := []string{"views.HomeView", "SpectacularSwaggerView", "SpectacularRedocView", "api-auth/", "api/v1/", "library/"}
	for _, fragment := range required {
		if !strings.Contains(full, fragment) {
			t.Fatalf("full URL config missing %q, got:\n%s", fragment, full)
		}
	}
}

func TestDiscoverPythonOptionsFromCandidatesDeduplicatesAndPrefersDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell scripts")
	}

	dir := t.TempDir()
	defaultPython := writeFakePythonExecutable(t, dir, "python-default", "3.12.3")
	newerPython := writeFakePythonExecutable(t, dir, "python-newer", "3.14.3")
	aliasPath := filepath.Join(dir, "python-alias")
	if err := os.Symlink(newerPython, aliasPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	options := ui.DiscoverPythonOptionsFromCandidatesForTest(
		[]string{newerPython, defaultPython, aliasPath, newerPython},
		defaultPython,
	)
	normalizedDefault, ok := ui.NormalizeExecutablePathForTest(defaultPython)
	if !ok {
		t.Fatalf("expected default test executable to normalize")
	}
	normalizedNewer, ok := ui.NormalizeExecutablePathForTest(newerPython)
	if !ok {
		t.Fatalf("expected newer test executable to normalize")
	}

	if len(options) != 2 {
		t.Fatalf("expected 2 unique Python options, got %d", len(options))
	}
	if options[0].Path != normalizedDefault {
		t.Fatalf("expected default Python to be first, got %s", options[0].Path)
	}
	if !options[0].IsDefault {
		t.Fatalf("expected first option to be marked default")
	}
	if options[1].Path != normalizedNewer {
		t.Fatalf("expected non-default newer interpreter second, got %s", options[1].Path)
	}
}

func TestInjectEnvConfigPostgresDockerRemovesSQLite(t *testing.T) {
	// Simulate a real Django startproject settings.py with the standard SQLite DATABASES block
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

	model := &ui.Model{ProjectConfig: ui.ProjectConfig{
		ProjectName:  "myproject",
		DatabaseType: "postgres-docker",
	}}

	result := ui.InjectEnvConfigForTest(fakeSettings, model)

	// Must have exactly ONE DATABASES = block
	count := strings.Count(result, "DATABASES =")
	if count != 1 {
		t.Fatalf("expected exactly 1 DATABASES = block, got %d.\nResult:\n%s", count, result)
	}

	// Must use postgresql backend
	if !strings.Contains(result, "django.db.backends.postgresql") {
		t.Fatalf("expected postgresql backend in result:\n%s", result)
	}

	// Must NOT contain sqlite3
	if strings.Contains(result, "sqlite3") {
		t.Fatalf("expected no sqlite3 references in result:\n%s", result)
	}

	// DATABASES must appear AFTER env = environ.Env()
	envPos := strings.Index(result, "env = environ.Env()")
	dbPos := strings.Index(result, "DATABASES =")
	if envPos < 0 || dbPos < 0 || dbPos < envPos {
		t.Fatalf("expected DATABASES to appear after env = environ.Env()\nenv pos=%d, db pos=%d\nResult:\n%s", envPos, dbPos, result)
	}

	// Assert no duplicate imports or settings
	duplicateChecks := map[string]int{
		"from pathlib import Path": 1,
		"import os":                0, // Django default has it; we don't add it
		"BASE_DIR =":               1,
		"SECRET_KEY =":             1,
		"DEBUG =":                  1,
		"ALLOWED_HOSTS =":          1,
	}
	for pattern, maxCount := range duplicateChecks {
		count := strings.Count(result, pattern)
		if pattern == "import os" {
			// import os may appear once (Django default) — just ensure not duplicated
			if count > 1 {
				t.Fatalf("expected at most 1 %q, got %d.\nResult:\n%s", pattern, count, result)
			}
		} else if count != maxCount {
			t.Fatalf("expected exactly %d %q, got %d.\nResult:\n%s", maxCount, pattern, count, result)
		}
	}
}

func TestInjectEnvConfigIsIdempotentAcrossDatabaseTypes(t *testing.T) {
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
		databaseType string
		engine       string
	}{
		{name: "sqlite", databaseType: "sqlite", engine: "django.db.backends.sqlite3"},
		{name: "postgres", databaseType: "postgres-docker", engine: "django.db.backends.postgresql"},
		{name: "mysql", databaseType: "mysql-docker", engine: "django.db.backends.mysql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &ui.Model{ProjectConfig: ui.ProjectConfig{DatabaseType: tt.databaseType}}
			once := ui.InjectEnvConfigForTest(baseSettings, model)
			twice := ui.InjectEnvConfigForTest(once, model)

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

func TestUpdateAdvancesFromProjectNameToPythonToDjangoSelection(t *testing.T) {
	restoreLoaders := ui.SetPickerLoadersForTest(
		func() []ui.PythonOption {
			return []ui.PythonOption{
				{
					Path:      "/opt/homebrew/bin/python3",
					Version:   "3.14.3",
					Label:     "Python 3.14.3 (/opt/homebrew/bin/python3) [default]",
					IsDefault: true,
					Major:     3,
					Minor:     14,
					Patch:     3,
				},
			}
		},
		func(ui.PythonOption) ([]ui.DjangoOption, string) {
			return []ui.DjangoOption{
				{
					Version:       "6.0.3",
					Label:         "6.0.3 [recommended] - compatible with Python 3.14",
					Recommended:   true,
					Compatibility: "compatible with Python 3.14",
				},
			}, "Showing the 5 latest compatible stable Django releases from PyPI."
		},
	)
	defer restoreLoaders()

	model := ui.NewModel()
	model.Step = ui.StepProjectName
	model.ProjectName = "django-stack"
	model.ProjectNameForm.State = huh.StateCompleted

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	current := updatedModel.(*ui.Model)
	if current.Step != ui.StepPythonSelection {
		t.Fatalf("expected StepPythonSelection after project name, got %v", current.Step)
	}
	if current.SelectedPythonPath != "/opt/homebrew/bin/python3" {
		t.Fatalf("expected default Python to be preselected, got %q", current.SelectedPythonPath)
	}

	current.PythonForm.State = huh.StateCompleted
	updatedModel, _ = current.Update(tea.KeyMsg{Type: tea.KeyEnter})
	current = updatedModel.(*ui.Model)
	if current.Step != ui.StepDjangoVersion {
		t.Fatalf("expected StepDjangoVersion after Python selection, got %v", current.Step)
	}
	if current.DjangoVersion != "6.0.3" {
		t.Fatalf("expected Django version to be preselected from compatible options, got %q", current.DjangoVersion)
	}
}

func TestCreateVSCodeFilesWritesTopLevelCompoundsAndFolderOpenTask(t *testing.T) {
	projectPath := t.TempDir()
	ui.CreateVSCodeFiles(projectPath, true)

	tasksBytes, err := os.ReadFile(filepath.Join(projectPath, ".vscode", "tasks.json"))
	if err != nil {
		t.Fatalf("failed to read tasks.json: %v", err)
	}
	tasksJSON := string(tasksBytes)
	if !strings.Contains(tasksJSON, "\"label\": \"Workspace: Start Dev\"") {
		t.Fatalf("expected composite workspace task in tasks.json, got:\n%s", tasksJSON)
	}
	if !strings.Contains(tasksJSON, "\"runOn\": \"folderOpen\"") {
		t.Fatalf("expected folderOpen auto-run in tasks.json, got:\n%s", tasksJSON)
	}

	launchBytes, err := os.ReadFile(filepath.Join(projectPath, ".vscode", "launch.json"))
	if err != nil {
		t.Fatalf("failed to read launch.json: %v", err)
	}
	launchJSON := string(launchBytes)
	if !strings.Contains(launchJSON, "\"compounds\"") {
		t.Fatalf("expected top-level compounds in launch.json, got:\n%s", launchJSON)
	}
	if strings.Contains(launchJSON, "\"configurations\": [{\"name\": \"Django + Tailwind\"") {
		t.Fatalf("compound should not be nested inside configurations, got:\n%s", launchJSON)
	}

	settingsBytes, err := os.ReadFile(filepath.Join(projectPath, ".vscode", "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	if !strings.Contains(string(settingsBytes), "\"task.allowAutomaticTasks\": \"on\"") {
		t.Fatalf("expected automatic task setting enabled, got:\n%s", string(settingsBytes))
	}

	welcomeBytes, err := os.ReadFile(filepath.Join(projectPath, "WELCOME.md"))
	if err != nil {
		t.Fatalf("failed to read WELCOME.md: %v", err)
	}
	welcome := string(welcomeBytes)
	if !strings.Contains(welcome, "SUPERUSER_SETUP.md") {
		t.Fatalf("expected WELCOME.md to mention superuser setup guide, got:\n%s", welcome)
	}
	if !strings.Contains(welcome, ".venv") {
		t.Fatalf("expected WELCOME.md to mention venv-based manage.py usage, got:\n%s", welcome)
	}
}

func TestGenerateStartScriptOpensVSCodeWithoutStartingDuplicateServers(t *testing.T) {
	script := ui.GenerateStartScriptForTest("/tmp/demo", "/usr/bin/code", true)
	if !strings.Contains(script, "code") || !strings.Contains(script, "Workspace: Start Dev") {
		t.Fatalf("expected script to explain VS Code workspace task startup, got:\n%s", script)
	}
	if strings.Contains(script, "nohup ./.venv/bin/python manage.py runserver") || strings.Contains(script, "nohup npm run watch:css") {
		t.Fatalf("script should not start duplicate background processes anymore, got:\n%s", script)
	}
}

func TestBuildScaffoldFormattingPlanIncludesMandatoryPackagesAndTemplateFiles(t *testing.T) {
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

	packages, commands, templateFiles, err := ui.BuildScaffoldFormattingPlanForTest(projectPath)
	if err != nil {
		t.Fatalf("BuildScaffoldFormattingPlanForTest failed: %v", err)
	}

	if len(packages) != 2 || packages[0] != "ruff" || packages[1] != "djlint" {
		t.Fatalf("expected mandatory formatter packages [ruff djlint], got %#v", packages)
	}
	if len(commands) != 2 {
		t.Fatalf("expected two mandatory Python formatting commands, got %#v", commands)
	}
	if commands[0].Module != "ruff" || strings.Join(commands[0].Args, " ") != "check --fix ." {
		t.Fatalf("unexpected first formatting command: %#v", commands[0])
	}
	if commands[1].Module != "ruff" || strings.Join(commands[1].Args, " ") != "format ." {
		t.Fatalf("unexpected second formatting command: %#v", commands[1])
	}
	if len(templateFiles) != 2 {
		t.Fatalf("expected two template files for djlint, got %#v", templateFiles)
	}
}

func TestBuildScaffoldFormattingPlanSkipsTemplatesWhenNoneExist(t *testing.T) {
	projectPath := t.TempDir()

	_, _, templateFiles, err := ui.BuildScaffoldFormattingPlanForTest(projectPath)
	if err != nil {
		t.Fatalf("BuildScaffoldFormattingPlanForTest failed: %v", err)
	}
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

	model := &ui.Model{
		ProjectConfig: ui.ProjectConfig{
			SetupRuff:      false,
			SetupPreCommit: false,
		},
	}

	var installedPackages []string
	restoreInstall := ui.SetFormattingInstallHookForTest(func(_ *ui.Model, _ string, packages ...string) error {
		installedPackages = append([]string{}, packages...)
		return nil
	})
	defer restoreInstall()

	var called []string
	restoreRunner := ui.SetPythonModuleCommandRunnerForTest(func(projectPath, module string, args ...string) ([]byte, error) {
		called = append(called, formatCommandKey(projectPath, module, args...))
		return []byte("ok"), nil
	})
	defer restoreRunner()

	if err := ui.RunMandatoryScaffoldFormattingForTest(model, projectPath); err != nil {
		t.Fatalf("RunMandatoryScaffoldFormattingForTest failed: %v", err)
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
}

func TestRunMandatoryScaffoldFormattingFailsOnRuffAutofixError(t *testing.T) {
	projectPath := t.TempDir()
	model := &ui.Model{}

	restoreInstall := ui.SetFormattingInstallHookForTest(func(_ *ui.Model, _ string, _ ...string) error {
		return nil
	})
	defer restoreInstall()

	restoreRunner := ui.SetPythonModuleCommandRunnerForTest(func(_ string, module string, args ...string) ([]byte, error) {
		if module == "ruff" && len(args) >= 2 && args[0] == "check" && args[1] == "--fix" {
			return []byte("E501 line too long"), errors.New("exit status 1")
		}
		return []byte("ok"), nil
	})
	defer restoreRunner()

	err := ui.RunMandatoryScaffoldFormattingForTest(model, projectPath)
	if err == nil {
		t.Fatalf("expected formatting failure when ruff autofix fails")
	}
	if !strings.Contains(err.Error(), "ruff autofix") {
		t.Fatalf("expected error to mention ruff autofix stage, got %v", err)
	}
	if strings.Contains(strings.Join(model.StepMessages, "\n"), "Generated code and templates formatted.") {
		t.Fatalf("success message should not be appended on formatting failure")
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

	model := &ui.Model{}

	restoreInstall := ui.SetFormattingInstallHookForTest(func(_ *ui.Model, _ string, _ ...string) error {
		return nil
	})
	defer restoreInstall()

	restoreRunner := ui.SetPythonModuleCommandRunnerForTest(func(_ string, module string, _ ...string) ([]byte, error) {
		if module == "djlint" {
			return []byte("template parse error"), errors.New("exit status 1")
		}
		return []byte("ok"), nil
	})
	defer restoreRunner()

	err := ui.RunMandatoryScaffoldFormattingForTest(model, projectPath)
	if err == nil {
		t.Fatalf("expected formatting failure when djlint fails")
	}
	if !strings.Contains(err.Error(), "template formatting") {
		t.Fatalf("expected error to mention template formatting stage, got %v", err)
	}
}

func TestGeneratedBookViewsUseGetObjectOr404ForStaleIDs(t *testing.T) {
	for _, htmx := range []bool{false, true} {
		content := ui.GenerateBookViewsForTest("library", htmx)
		if !strings.Contains(content, "get_object_or_404") {
			t.Fatalf("expected generated views to import/use get_object_or_404 when htmx=%v, got:\n%s", htmx, content)
		}
		if strings.Contains(content, "Book.objects.get(id=book_id)") {
			t.Fatalf("generated views should not use direct Book.objects.get for edit/delete when htmx=%v, got:\n%s", htmx, content)
		}
	}
}

func TestPythonToolingMetadataMatchesSelectedInterpreter(t *testing.T) {
	tests := []struct {
		name            string
		option          ui.PythonOption
		wantRequires    string
		wantRuffTarget  string
		wantMypyVersion string
	}{
		{
			name:            "python_38",
			option:          ui.PythonOption{Path: "/tmp/python38", Version: "3.8.18", Major: 3, Minor: 8, Patch: 18},
			wantRequires:    ">=3.8",
			wantRuffTarget:  "py38",
			wantMypyVersion: "3.8",
		},
		{
			name:            "python_312",
			option:          ui.PythonOption{Path: "/tmp/python312", Version: "3.12.4", Major: 3, Minor: 12, Patch: 4},
			wantRequires:    ">=3.12",
			wantRuffTarget:  "py312",
			wantMypyVersion: "3.12",
		},
		{
			name:            "python_314",
			option:          ui.PythonOption{Path: "/tmp/python314", Version: "3.14.3", Major: 3, Minor: 14, Patch: 3},
			wantRequires:    ">=3.14",
			wantRuffTarget:  "py314",
			wantMypyVersion: "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &ui.Model{
				ProjectConfig: ui.ProjectConfig{
					SelectedPythonPath:    tt.option.Path,
					SelectedPythonVersion: tt.option.Version,
				},
				AvailablePythonOptions: []ui.PythonOption{tt.option},
			}

			requires, ruffTarget, mypyVersion := ui.PythonToolingMetadataForTest(model)
			if requires != tt.wantRequires || ruffTarget != tt.wantRuffTarget || mypyVersion != tt.wantMypyVersion {
				t.Fatalf("unexpected tooling metadata for %s: got (%s, %s, %s), want (%s, %s, %s)", tt.name, requires, ruffTarget, mypyVersion, tt.wantRequires, tt.wantRuffTarget, tt.wantMypyVersion)
			}
		})
	}
}

func TestFindAvailablePortRespectsPreferredPortRange(t *testing.T) {
	restore := ui.SetPortListenerForTest(func(network, address string) (net.Listener, error) {
		switch address {
		case "127.0.0.1:4500":
			return nil, errors.New("address already in use")
		case "127.0.0.1:4501":
			return fakeListener{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4501}}, nil
		default:
			return nil, errors.New("unexpected address: " + address)
		}
	})
	defer restore()

	port, err := ui.FindAvailablePortForTest(4500)
	if err != nil {
		t.Fatalf("FindAvailablePortForTest failed: %v", err)
	}

	foundPort, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("expected numeric port, got %q", port)
	}
	if foundPort != 4501 {
		t.Fatalf("expected next available port 4501, got %d", foundPort)
	}
}

func TestFindAvailablePortUsesDatabaseSpecificDefaults(t *testing.T) {
	for _, preferred := range []int{3306, 5432} {
		restore := ui.SetPortListenerForTest(func(network, address string) (net.Listener, error) {
			if address != "127.0.0.1:"+strconv.Itoa(preferred) {
				return nil, errors.New("unexpected address: " + address)
			}
			return fakeListener{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: preferred}}, nil
		})
		port, err := ui.FindAvailablePortForTest(preferred)
		restore()
		if err != nil {
			t.Fatalf("FindAvailablePortForTest(%d) failed: %v", preferred, err)
		}
		foundPort, err := strconv.Atoi(port)
		if err != nil {
			t.Fatalf("expected numeric port for preferred %d, got %q", preferred, port)
		}
		if foundPort < preferred {
			t.Fatalf("expected port >= %d, got %d", preferred, foundPort)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type fakeListener struct {
	addr net.Addr
}

func (f fakeListener) Accept() (net.Conn, error) {
	return nil, errors.New("not implemented")
}

func (f fakeListener) Close() error {
	return nil
}

func (f fakeListener) Addr() net.Addr {
	return f.addr
}

func writeFakePythonExecutable(t *testing.T, dir, name, version string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  echo \"Python " + version + "\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"-c\" ]; then\n" +
		"  case \"$2\" in\n" +
		"    *version_info.micro*) echo \"" + version + "\" ;;\n" +
		"    *print*ok*) echo \"ok\" ;;\n" +
		"    *) echo \"" + version + "\" ;;\n" +
		"  esac\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo \"unsupported\" >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write fake python executable: %v", err)
	}
	return path
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

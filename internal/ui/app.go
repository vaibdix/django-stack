package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (m *Model) CreateDjangoApp(projectPath, settingsPath string) error {
	if m.AppName == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	pythonVenvPath := getPythonPath(projectPath)
	cmd := exec.CommandContext(ctx, pythonVenvPath, ManageFile, "startapp", m.AppName)

	cmd.Dir = projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create app '%s': %w\nOutput: %s", m.AppName, err, string(output))
	}

	appPath := filepath.Join(projectPath, m.AppName)

	if err := m.createAppsPy(appPath); err != nil {
		return fmt.Errorf("failed to create apps.py: %w", err)
	}

	if err := m.createAdminPy(appPath); err != nil {
		return fmt.Errorf("failed to create admin.py: %w", err)
	}

	if err := m.createTestsPy(appPath); err != nil {
		return fmt.Errorf("failed to create tests.py: %w", err)
	}

	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}
	if err := se.AddToList(InstalledApps, m.AppName); err != nil {
		return fmt.Errorf("failed to add app '%s' to %s: %w", m.AppName, InstalledApps, err)
	}
	if err := se.Save(); err != nil {
		return err
	}
	m.AppendStepMessage(fmt.Sprintf("Created and registered Django app: %s", m.AppName))

	if m.CreateAppTemplates && !m.SetupRestFramework {
		if err := m.SetupAppTemplates(projectPath); err != nil {
			return err
		}
	}
	return nil
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (m *Model) createAppsPy(appPath string) error {
	appNameTitle := titleCase(m.AppName)
	appsContent := fmt.Sprintf(`from django.apps import AppConfig


class %sConfig(AppConfig):
    default_auto_field = 'django.db.models.BigAutoField'
    name = '%s'
    verbose_name = '%s'
`, appNameTitle, m.AppName, appNameTitle)

	appsPath := filepath.Join(appPath, "apps.py")
	if err := os.WriteFile(appsPath, []byte(appsContent), 0644); err != nil {
		return err
	}
	m.AppendStepMessage("Created apps.py with AppConfig.")
	return nil
}

func (m *Model) createAdminPy(appPath string) error {
	adminContent := m.adminPyContent()

	adminPath := filepath.Join(appPath, "admin.py")
	if err := os.WriteFile(adminPath, []byte(adminContent), 0644); err != nil {
		return err
	}
	m.AppendStepMessage("Created admin.py for model registration.")
	return nil
}

func (m *Model) adminPyContent() string {
	if m.SetupRestFramework && m.AppName != "" {
		return `from django.contrib import admin
from .models import Book


@admin.register(Book)
class BookAdmin(admin.ModelAdmin):
    list_display = ("title", "author", "isbn", "publication_date", "price", "created_at")
    search_fields = ("title", "author", "isbn")
    list_filter = ("publication_date", "created_at", "updated_at")
    ordering = ("-created_at",)
    readonly_fields = ("created_at", "updated_at")
`
	}

	return `from django.contrib import admin

# Register your models here.
# Example:
# from .models import YourModel
# admin.site.register(YourModel)
`
}

func (m *Model) createTestsPy(appPath string) error {
	testsContent := fmt.Sprintf(`from django.test import TestCase

# Create your tests here.
class %sTestCase(TestCase):
    def setUp(self):
        pass

    def test_example(self):
        self.assertTrue(True)
`, titleCase(m.AppName))

	testsPath := filepath.Join(appPath, "tests.py")
	if err := os.WriteFile(testsPath, []byte(testsContent), 0644); err != nil {
		return err
	}
	m.AppendStepMessage("Created tests.py with basic test structure.")
	return nil
}

func (m *Model) SetupAppTemplates(projectPath string) error {
	appPath := filepath.Join(projectPath, m.AppName)
	appTemplatesDir := filepath.Join(appPath, "templates", m.AppName)
	if err := os.MkdirAll(appTemplatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create app templates directory %s: %w",appTemplatesDir, err)
	}

	appIndexContent := `{% extends 'base.html' %}
{% block title %}` + titleCase(m.AppName) + ` Home{% endblock %}
{% block content %}<h1>Welcome to the ` + m.AppName + `</h1>{% endblock %}`
	if _, err := os.Stat(filepath.Join(projectPath, "templates", "base.html")); os.IsNotExist(err) {
		appIndexContent = `<!DOCTYPE html><html><head><title>` + titleCase(m.AppName) + `</title></head><body><h1>Welcome to the ` + m.AppName + ` app!</h1></body></html>`
	}
	if err := os.WriteFile(filepath.Join(appTemplatesDir, "index.html"), []byte(appIndexContent), 0644); err != nil {
		return fmt.Errorf("failed to create index.html for app %s: %w", m.AppName, err)
	}

	viewsContent := fmt.Sprintf(`from django.shortcuts import render

def index(request):
    return render(request, '`+m.AppName+`/index.html')
`, m.AppName)
	if err := os.WriteFile(filepath.Join(appPath, "views.py"), []byte(viewsContent), 0644); err != nil {
		return fmt.Errorf("failed to create views.py for app %s: %w", m.AppName, err)
	}

	appUrlsContent := fmt.Sprintf(`from django.urls import path
from . import views

app_name = '%s'
urlpatterns = [
    path('', views.index, name='index'),
]
`, m.AppName)
	if err := os.WriteFile(filepath.Join(appPath, URLsFile), []byte(appUrlsContent), 0644); err != nil {
		return fmt.Errorf("failed to create urls.py for app %s: %w", m.AppName, err)
	}
	m.AppendStepMessage(fmt.Sprintf("Configured templates, views, and URLs for app: %s", m.AppName))
	return nil
}

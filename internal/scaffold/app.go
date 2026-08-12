package scaffold

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func (e *Env) createDjangoApp(projectPath, settingsPath string) error {
	if e.Cfg.AppName == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	pythonVenvPath := pyenv.InterpreterPath(projectPath)
	cmd := exec.CommandContext(ctx, pythonVenvPath, ManageFile, "startapp", e.Cfg.AppName)

	cmd.Dir = projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create app '%s': %w\nOutput: %s", e.Cfg.AppName, err, string(output))
	}

	appPath := filepath.Join(projectPath, e.Cfg.AppName)

	if err := e.createAppsPy(appPath); err != nil {
		return fmt.Errorf("failed to create apps.py: %w", err)
	}

	if err := e.createAdminPy(appPath); err != nil {
		return fmt.Errorf("failed to create admin.py: %w", err)
	}

	if err := e.createTestsPy(appPath); err != nil {
		return fmt.Errorf("failed to create tests.py: %w", err)
	}

	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}
	if err := se.AddToList(InstalledApps, e.Cfg.AppName); err != nil {
		return fmt.Errorf("failed to add app '%s' to %s: %w", e.Cfg.AppName, InstalledApps, err)
	}
	if err := se.Save(); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Created and registered Django app: %s", e.Cfg.AppName))

	if e.Cfg.CreateAppTemplates && !e.Cfg.SetupRestFramework {
		if err := e.applyAppTemplates(projectPath); err != nil {
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

type appNames struct {
	AppName      string
	AppNameTitle string
}

func (e *Env) appNames() appNames {
	return appNames{AppName: e.Cfg.AppName, AppNameTitle: titleCase(e.Cfg.AppName)}
}

func (e *Env) createAppsPy(appPath string) error {
	appsContent := render("apps.py.tmpl", e.appNames())

	appsPath := filepath.Join(appPath, "apps.py")
	if err := os.WriteFile(appsPath, []byte(appsContent), 0644); err != nil {
		return err
	}
	e.Step("Created apps.py with AppConfig.")
	return nil
}

func (e *Env) createAdminPy(appPath string) error {
	adminContent := e.adminPyContent()

	adminPath := filepath.Join(appPath, "admin.py")
	if err := os.WriteFile(adminPath, []byte(adminContent), 0644); err != nil {
		return err
	}
	e.Step("Created admin.py for model registration.")
	return nil
}

func (e *Env) adminPyContent() string {
	if e.Cfg.SetupRestFramework && e.Cfg.AppName != "" {
		return renderStatic("admin_book.py.tmpl")
	}
	return renderStatic("admin_plain.py.tmpl")
}

func (e *Env) testsPyContent() string {
	if e.Cfg.SetupRestFramework && e.Cfg.AppName != "" {
		if e.Cfg.APIOnly() {
			return render("tests_api.py.tmpl", e.appNames())
		}
		return render("tests_book.py.tmpl", e.appNames())
	}
	return render("tests_plain.py.tmpl", e.appNames())
}

func (e *Env) createTestsPy(appPath string) error {
	testsContent := e.testsPyContent()

	testsPath := filepath.Join(appPath, "tests.py")
	if err := os.WriteFile(testsPath, []byte(testsContent), 0644); err != nil {
		return err
	}
	e.Step("Created tests.py with basic test structure.")
	return nil
}

func (e *Env) applyAppTemplates(projectPath string) error {
	appPath := filepath.Join(projectPath, e.Cfg.AppName)
	appTemplatesDir := filepath.Join(appPath, "templates", e.Cfg.AppName)
	if err := os.MkdirAll(appTemplatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create app templates directory %s: %w", appTemplatesDir, err)
	}

	appIndexContent := `{% extends 'base.html' %}
{% block title %}` + titleCase(e.Cfg.AppName) + ` Home{% endblock %}
{% block content %}<h1>Welcome to the ` + e.Cfg.AppName + `</h1>{% endblock %}`
	if _, err := os.Stat(filepath.Join(projectPath, "templates", "base.html")); os.IsNotExist(err) {
		appIndexContent = `<!DOCTYPE html><html><head><title>` + titleCase(e.Cfg.AppName) + `</title></head><body><h1>Welcome to the ` + e.Cfg.AppName + ` app!</h1></body></html>`
	}
	if err := os.WriteFile(filepath.Join(appTemplatesDir, "index.html"), []byte(appIndexContent), 0644); err != nil {
		return fmt.Errorf("failed to create index.html for app %s: %w", e.Cfg.AppName, err)
	}

	viewsContent := fmt.Sprintf(`from django.http import HttpRequest
from django.http import HttpResponse
from django.shortcuts import render

def index(request: HttpRequest) -> HttpResponse:
    return render(request, '%s/index.html')
`, e.Cfg.AppName)
	if err := os.WriteFile(filepath.Join(appPath, "views.py"), []byte(viewsContent), 0644); err != nil {
		return fmt.Errorf("failed to create views.py for app %s: %w", e.Cfg.AppName, err)
	}

	appUrlsContent := fmt.Sprintf(`from django.urls import path
from . import views

app_name = '%s'
urlpatterns = [
    path('', views.index, name='index'),
]
`, e.Cfg.AppName)
	if err := os.WriteFile(filepath.Join(appPath, URLsFile), []byte(appUrlsContent), 0644); err != nil {
		return fmt.Errorf("failed to create urls.py for app %s: %w", e.Cfg.AppName, err)
	}
	e.Step(fmt.Sprintf("Configured templates, views, and URLs for app: %s", e.Cfg.AppName))
	return nil
}

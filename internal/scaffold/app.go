package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-stack/internal/scaffold/pyenv"
)

func (e *Env) createDjangoApp(projectPath, settingsPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	pythonVenvPath := pyenv.InterpreterPath(projectPath)
	if _, err := e.runIn(ctx, projectPath, pythonVenvPath, ManageFile, "startapp", e.Cfg.AppName); err != nil {
		return fmt.Errorf("failed to create app '%s': %w", e.Cfg.AppName, err)
	}

	appPath := filepath.Join(projectPath, e.Cfg.AppName)

	if err := e.createAppsPy(appPath); err != nil {
		return fmt.Errorf("failed to create apps.py: %w", err)
	}

	if err := e.writeBookApp(projectPath); err != nil {
		return err
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
	e.Step("Created and registered Django app: " + e.Cfg.AppName)
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

type testsData struct {
	appNames
	HTMX bool
	Ruff bool
}

func (e *Env) testsData() testsData {
	return testsData{appNames: e.appNames(), HTMX: e.Cfg.SetupHTMX, Ruff: e.Cfg.SetupRuff}
}

func (e *Env) apiTestsContent() string {
	if e.Cfg.SetupNinja {
		return render("tests_ninja.py.tmpl", e.testsData())
	}
	return render("tests_api.py.tmpl", e.testsData())
}

func (e *Env) testFiles() map[string]string {
	files := map[string]string{
		filepath.Join("tests", "__init__.py"):    "",
		filepath.Join("tests", "test_models.py"): render("tests_model.py.tmpl", e.appNames()),
	}
	if e.hasBookPages() {
		files[filepath.Join("tests", "test_pages.py")] = render("tests_book.py.tmpl", e.testsData())
	}
	if e.Cfg.HasAPI() {
		files[filepath.Join("tests", "test_api.py")] = e.apiTestsContent()
	}
	return files
}

func (e *Env) createTestsPy(appPath string) error {
	if err := os.Remove(filepath.Join(appPath, "tests.py")); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Join(appPath, "tests"), 0755); err != nil {
		return err
	}
	for name, content := range e.testFiles() {
		if err := os.WriteFile(filepath.Join(appPath, name), []byte(content), 0644); err != nil {
			return err
		}
	}
	e.Step("Created the app's tests.")
	return nil
}

package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func (e *Env) createDjangoApp(projectPath, settingsPath string) error {
	if e.Cfg.AppName == "" {
		return nil
	}

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
	e.Step(fmt.Sprintf("Created and registered Django app: %s", e.Cfg.AppName))
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

func (e *Env) testsPyContent() string {
	switch {
	case e.hasBookPages():
		return render("tests_book.py.tmpl", e.appNames())
	case e.Cfg.SetupNinja:
		return render("tests_ninja.py.tmpl", e.appNames())
	case e.Cfg.SetupRestFramework:
		return render("tests_api.py.tmpl", e.appNames())
	default:
		return render("tests_model.py.tmpl", e.appNames())
	}
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

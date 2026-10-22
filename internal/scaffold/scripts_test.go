package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/pyenv"
)

func TestDevServerCommandUsesVenvInterpreterOnly(t *testing.T) {
	t.Parallel()

	if command := devServerCommandFor(pyenv.Windows); command != `.venv\Scripts\python.exe manage.py runserver` {
		t.Fatalf("unexpected Windows dev server command: %q", command)
	}
	command := devServerCommandFor("linux")
	if command != "./.venv/bin/python manage.py runserver" {
		t.Fatalf("unexpected Unix dev server command: %q", command)
	}
	if strings.Contains(command, "uv run") {
		t.Fatalf("dev server command should not depend on uv: %q", command)
	}
}

func TestGenerateStartScriptOpensVSCodeWithoutStartingDuplicateServers(t *testing.T) {
	t.Parallel()

	script := generateStartScript("linux", true)
	if !strings.Contains(script, "code") || !strings.Contains(script, "Workspace: Start Dev") {
		t.Fatalf("expected script to explain VS Code workspace task startup, got:\n%s", script)
	}
	if strings.Contains(script, "nohup ./.venv/bin/python manage.py runserver") || strings.Contains(script, "nohup npm run watch:css") {
		t.Fatalf("script should not start duplicate background processes anymore, got:\n%s", script)
	}
}

package scaffold

import (
	"context"
	"os/exec"
	"runtime"

	"github.com/vaibdix/django-stack/internal/pyenv"
)

func (e *Env) findVSCode() string {
	path, err := exec.LookPath("code")
	if err != nil {
		return ""
	}
	return path
}

func (e *Env) StartDevelopmentEnvironment() {
	projectPath, err := e.projectPath()
	if err != nil {
		e.Step("Could not open the project: " + err.Error())
		return
	}

	editor := e.findVSCode()
	if editor == "" {
		e.Step("VS Code was not found. Open " + projectPath + " in your editor, or run " + devStartScriptName() + " there.")
		return
	}

	cmd := exec.CommandContext(context.Background(), editor, "-n", projectPath)
	cmd.Dir = projectPath
	if err := cmd.Start(); err != nil {
		e.Step("Failed to open VS Code: " + err.Error())
		e.Step("Run " + devStartScriptName() + " in the project folder")
		return
	}
	go func() { _ = cmd.Wait() }()
	e.Step("VS Code opened; it starts the dev task once you allow automatic tasks.")
}

func generateStartScript(goos string, useTailwind bool) string {
	return render(devStartScriptNameFor(goos)+".tmpl", struct{ Tailwind bool }{Tailwind: useTailwind})
}

func devStartScriptName() string { return devStartScriptNameFor(runtime.GOOS) }

func devStartScriptNameFor(goos string) string { return "start-dev" + pyenv.ScriptExt(goos) }

func devServerCommandFor(goos string) string {
	return pyenv.InterpreterCommand(goos) + " manage.py runserver"
}

package scaffold

import (
	"context"
	"os/exec"
	"runtime"
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

func generateStartScript(windows, useTailwind bool) string {
	data := struct{ Tailwind bool }{Tailwind: useTailwind}
	if windows {
		return render("start-dev.bat.tmpl", data)
	}
	return render("start-dev.sh.tmpl", data)
}

func devStartScriptName() string { return devStartScriptNameFor(runtime.GOOS == "windows") }

func devStartScriptNameFor(windows bool) string {
	if windows {
		return "start-dev.bat"
	}
	return "start-dev.sh"
}

func devServerCommandFor(windows bool) string {
	if windows {
		return `.venv\Scripts\python.exe manage.py runserver`
	}
	return "./.venv/bin/python manage.py runserver"
}

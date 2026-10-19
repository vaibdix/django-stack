package scaffold

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const lookupTimeout = 5 * time.Second

func (e *Env) findVSCode() string {
	if runtime.GOOS != "windows" {
		if path, err := exec.LookPath("code"); err == nil {
			return path
		}
		return ""
	}

	paths := []string{
		`C:\Program Files\Microsoft VS Code Insiders\Code.exe`,
		`C:\Program Files (x86)\Microsoft VS Code Insiders\Code.exe`,
		`C:\Program Files\Microsoft VS Code\Code.exe`,
		`C:\Program Files (x86)\Microsoft VS Code\Code.exe`,
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Programs\Microsoft VS Code\Code.exe`),
		filepath.Join(os.Getenv("APPDATA"), `..\Local\Programs\Microsoft VS Code\Code.exe`),
		filepath.Join(os.Getenv("USERPROFILE"), `AppData\Local\Programs\Microsoft VS Code\Code.exe`),
		filepath.Join(os.Getenv("ProgramFiles"), `Microsoft VS Code\Code.exe`),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), `Microsoft VS Code\Code.exe`),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	ctx, cancel := context.WithTimeout(e.context(), lookupTimeout)
	defer cancel()

	if output, err := e.runner().Run(ctx, "", "where", "code"); err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			return strings.TrimSpace(lines[0])
		}
	}

	return ""
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

	cmd := exec.CommandContext(context.WithoutCancel(context.Background()), editor, "-n", projectPath)
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

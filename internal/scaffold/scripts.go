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

func findVSCode() string {
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

	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "where", "code")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			return strings.TrimSpace(lines[0])
		}
	}

	return ""
}

func (e *Env) StartDevelopmentEnvironment() {
	projectPath := e.Cfg.ProjectName
	if !filepath.IsAbs(projectPath) {
		wd, _ := os.Getwd()
		projectPath = filepath.Join(wd, e.Cfg.ProjectName)
	}

	scriptPath := filepath.Join(projectPath, devStartScriptName())

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		e.Step(filepath.Base(scriptPath) + " not found in project folder")
		e.Step("Navigate to " + projectPath + " and run it manually")
		return
	}

	e.Step("Running " + filepath.Base(scriptPath) + "...")

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(context.Background(), "cmd.exe", "/c", "start", "", scriptPath)
		cmd.Stdout = nil
		cmd.Stderr = nil
	} else {
		cmd = exec.CommandContext(context.Background(), "sh", scriptPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		e.Step("Failed to run script: " + err.Error())
		e.Step("Run " + filepath.Base(scriptPath) + " in the project folder")
	} else {
		e.Step("Development startup script launched.")
	}
}

type startScriptData struct {
	ProjectPath string
	VSCodePath  string
	Tailwind    bool
}

func generateStartScript(projectPath, vscodePath string, useTailwind bool) string {
	data := startScriptData{ProjectPath: projectPath, VSCodePath: vscodePath, Tailwind: useTailwind}
	if runtime.GOOS == "windows" {
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

func devServerCommand() string { return devServerCommandFor(runtime.GOOS == "windows") }

func devServerCommandFor(windows bool) string {
	if windows {
		return `.venv\Scripts\python.exe manage.py runserver`
	}
	return "./.venv/bin/python manage.py runserver"
}

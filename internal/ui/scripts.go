package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

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

	cmd := exec.Command("where", "code")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			return strings.TrimSpace(lines[0])
		}
	}

	return ""
}

func (m *Model) StartDevelopmentEnvironment() {
	projectPath := m.ProjectName
	if !filepath.IsAbs(projectPath) {
		wd, _ := os.Getwd()
		projectPath = filepath.Join(wd, m.ProjectName)
	}

	scriptPath := filepath.Join(projectPath, devStartScriptName())

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		m.AppendStepMessage(filepath.Base(scriptPath) + " not found in project folder")
		m.AppendStepMessage("Navigate to " + projectPath + " and run it manually")
		return
	}

	m.AppendStepMessage("Running " + filepath.Base(scriptPath) + "...")

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", "start", "", scriptPath)
		cmd.Stdout = nil
		cmd.Stderr = nil
	} else {
		cmd = exec.Command("sh", scriptPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		m.AppendStepMessage("Failed to run script: " + err.Error())
		m.AppendStepMessage("Run " + filepath.Base(scriptPath) + " in the project folder")
	} else {
		m.AppendStepMessage("Development startup script launched.")
	}
}

func generateStartScript(projectPath, vscodePath string, useTailwind bool) string {
	if runtime.GOOS == "windows" {
		return generateWindowsStartScript(projectPath, vscodePath, useTailwind)
	}
	return generateUnixStartScript(projectPath, vscodePath, useTailwind)
}

func devStartScriptName() string {
	if runtime.GOOS == "windows" {
		return "start-dev.bat"
	}
	return "start-dev.sh"
}

func devServerCommand() string {
	if runtime.GOOS == "windows" {
		return `.venv\Scripts\python.exe manage.py runserver`
	}
	return "./.venv/bin/python manage.py runserver"
}

func generateWindowsStartScript(projectPath, vscodePath string, useTailwind bool) string {
	s := `@echo off
echo ========================================
echo   Django Development Environment
echo ========================================
echo.
echo Opening VS Code...
echo.

echo [1/1] Opening VS Code...
`

	if vscodePath != "" {
		s += `start "" "` + vscodePath + `" -n "` + projectPath + `"
`
	} else {
		s += `where code >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    start "" code -n "` + projectPath + `"
) else (
    echo VS Code executable not found. Open the project manually if needed.
)
`
	}
	s += `
echo.
echo ========================================
echo   VS Code opened
echo ========================================
echo.
echo   The workspace auto-runs the "Workspace: Start Dev" task.
echo   Use the integrated terminals inside VS Code for Django and Tailwind.
echo.
pause
`
	return s
}

func generateUnixStartScript(projectPath, vscodePath string, useTailwind bool) string {
	var builder strings.Builder
	builder.WriteString("#!/bin/sh\n")
	builder.WriteString("set -eu\n\n")
	builder.WriteString("SCRIPT_DIR=\"$(CDPATH= cd -- \"$(dirname -- \"$0\")\" && pwd)\"\n")
	builder.WriteString("cd \"$SCRIPT_DIR\"\n\n")

	if vscodePath != "" {
		builder.WriteString("\"" + vscodePath + "\" -n \"$SCRIPT_DIR\" >/dev/null 2>&1 &\n")
	} else {
		builder.WriteString("if command -v code >/dev/null 2>&1; then\n")
		builder.WriteString("  code -n \"$SCRIPT_DIR\" >/dev/null 2>&1 &\n")
		builder.WriteString("fi\n")
	}

	builder.WriteString("echo \"VS Code opened in a new window.\"\n")
	builder.WriteString("echo \"The workspace auto-runs the 'Workspace: Start Dev' task in integrated terminals.\"\n")
	if useTailwind {
		builder.WriteString("echo \"Watch the Django and Tailwind terminals inside VS Code.\"\n")
	} else {
		builder.WriteString("echo \"Watch the Django terminal inside VS Code.\"\n")
	}
	builder.WriteString("echo \"Open http://localhost:8000 once the server is ready.\"\n")

	return builder.String()
}

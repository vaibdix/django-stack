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

// lookupTimeout bounds the Windows `where code` probe.
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

	// `where` is a fast local lookup, but it still gets a deadline so a wedged
	// filesystem cannot hang project generation indefinitely.
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

// StartDevelopmentEnvironment runs the generated start-dev script, which opens
// the project in VS Code and starts its dev servers.
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

	// This script is deliberately detached: it outlives the CLI, which exits as
	// soon as it has been launched. It therefore gets context.Background()
	// rather than the run-scoped context — binding it to the run would kill the
	// user's editor and dev server the moment the TUI closed.
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

// startScriptData drives both start-dev script templates.
//
// Tailwind only affects the Unix script's closing hint; the Windows script
// mentions both terminals unconditionally. That asymmetry was previously hidden
// behind a useTailwind parameter that generateWindowsStartScript accepted and
// never read.
type startScriptData struct {
	ProjectPath string
	VSCodePath  string
	Tailwind    bool
}

// generateStartScript returns the platform-appropriate dev startup script.
func generateStartScript(projectPath, vscodePath string, useTailwind bool) string {
	data := startScriptData{ProjectPath: projectPath, VSCodePath: vscodePath, Tailwind: useTailwind}
	if runtime.GOOS == "windows" {
		return render("start-dev.bat.tmpl", data)
	}
	return render("start-dev.sh.tmpl", data)
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

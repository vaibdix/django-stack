package ui

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func CreateVSCodeFiles(projectPath string, setupTailwind bool) {
	createVSCodeTasks(projectPath, setupTailwind)
}

func OpenVSCode(projectPath string) error {
	cmd := exec.Command("code", projectPath)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func createVSCodeTasks(projectPath string, setupTailwind bool) {
	vscodeDir := filepath.Join(projectPath, ".vscode")
	_ = os.MkdirAll(vscodeDir, 0755)

	presentation := map[string]interface{}{
		"group":  "dev",
		"reveal": "always",
		"panel":  "dedicated",
		"focus":  false,
		"clear":  false,
	}

	tasks := []map[string]interface{}{
		{
			"label":          "Django: Run server",
			"type":           "shell",
			"command":        devServerCommand(),
			"problemMatcher": []string{},
			"isBackground":   true,
			"presentation":   presentation,
		},
	}

	if runtime.GOOS == "windows" {
		tasks[0]["options"] = map[string]interface{}{
			"shell": map[string]interface{}{
				"executable": "cmd.exe",
				"args":       []string{"/c"},
			},
		}
	}

	if setupTailwind {
		tasks = append(tasks, map[string]interface{}{
			"label":          "Tailwind: Watch CSS",
			"type":           "shell",
			"command":        "npm run watch:css",
			"problemMatcher": []string{},
			"isBackground":   true,
			"presentation":   presentation,
		})
	}

	startTaskDependencies := []string{"Django: Run server"}
	if setupTailwind {
		startTaskDependencies = append(startTaskDependencies, "Tailwind: Watch CSS")
	}
	tasks = append(tasks, map[string]interface{}{
		"label":        "Workspace: Start Dev",
		"dependsOn":    startTaskDependencies,
		"dependsOrder": "parallel",
		"runOptions": map[string]interface{}{
			"runOn": "folderOpen",
		},
		"problemMatcher": []string{},
	})

	tasksObject := map[string]interface{}{
		"version": "2.0.0",
		"tasks":   tasks,
	}
	tasksJSON, _ := json.MarshalIndent(tasksObject, "", "  ")
	_ = os.WriteFile(filepath.Join(vscodeDir, "tasks.json"), tasksJSON, 0644)

	launchConfigs := []map[string]interface{}{
		{
			"name":       "Django: Run Server",
			"type":       "debugpy",
			"request":    "launch",
			"program":    "${workspaceFolder}/manage.py",
			"args":       []string{"runserver", "--noreload"},
			"django":     true,
			"justMyCode": true,
			"console":    "integratedTerminal",
		},
	}

	compounds := []map[string]interface{}{}
	if setupTailwind {
		launchConfigs = append(launchConfigs, map[string]interface{}{
			"name":              "Tailwind: Watch CSS",
			"type":              "node",
			"request":           "launch",
			"runtimeExecutable": "npm",
			"runtimeArgs":       []string{"run", "watch:css"},
			"console":           "integratedTerminal",
		})
		compounds = append(compounds, map[string]interface{}{
			"name":           "Django + Tailwind",
			"configurations": []string{"Django: Run Server", "Tailwind: Watch CSS"},
			"stopAll":        true,
		})
	}

	launchObject := map[string]interface{}{
		"version":        "0.2.0",
		"configurations": launchConfigs,
	}
	if len(compounds) > 0 {
		launchObject["compounds"] = compounds
	}
	launchJSON, _ := json.MarshalIndent(launchObject, "", "  ")
	_ = os.WriteFile(filepath.Join(vscodeDir, "launch.json"), launchJSON, 0644)

	extensionsContent := `{
  "recommendations": [
    "ms-python.python",
    "esbenp.prettier-vscode",
    "bradlc.vscode-tailwindcss"
  ]
}`
	_ = os.WriteFile(filepath.Join(vscodeDir, "extensions.json"), []byte(extensionsContent), 0644)

	settingsContent := `{
  "task.allowAutomaticTasks": "on"
}`
	if runtime.GOOS == "windows" {
		settingsContent = `{
  "task.allowAutomaticTasks": "on",
  "terminal.integrated.defaultProfile.windows": "Command Prompt"
}`
	}
	_ = os.WriteFile(filepath.Join(vscodeDir, "settings.json"), []byte(settingsContent), 0644)

	welcomeContent := "# Welcome to Your Django Project!\n\n"
	welcomeContent += "## Getting Started\n\n"
	if setupTailwind {
		welcomeContent += "- Opening this folder in VS Code auto-runs `Workspace: Start Dev`, which starts Django and Tailwind in grouped integrated terminals.\n"
		welcomeContent += "- You can manually rerun `Workspace: Start Dev` from the Tasks menu if needed.\n\n"
	} else {
		welcomeContent += "- Opening this folder in VS Code auto-runs `Workspace: Start Dev`, which starts the Django server in an integrated terminal.\n"
		welcomeContent += "- You can manually rerun `Workspace: Start Dev` from the Tasks menu if needed.\n\n"
	}
	welcomeContent += "## Dev Startup Script\n"
	welcomeContent += "- `" + devStartScriptName() + "` is generated in the project root.\n"
	welcomeContent += "- It opens a fresh VS Code window for the workspace so the auto-run task can start the dev processes.\n\n"
	welcomeContent += "## Admin Setup\n"
	welcomeContent += "- `" + superuserGuideFile + "` explains how to create a Django superuser manually.\n"
	welcomeContent += "- Prefer the project `.venv` commands from that guide for manual `manage.py` usage unless you have activated the virtual environment.\n\n"
	welcomeContent += "## Project Structure\n"
	welcomeContent += "- `manage.py` - Django management script\n"
	welcomeContent += "- `templates/` - HTML templates when enabled\n"
	welcomeContent += "- `static/` - Static files\n"
	welcomeContent += "- `" + filepath.Base(projectPath) + "/` - Django project configuration\n"
	_ = os.WriteFile(filepath.Join(projectPath, "WELCOME.md"), []byte(welcomeContent), 0644)
}

func (m *Model) SetupServerInstructions(projectPath string) {
	if m.RunServer {
		m.AppendStepMessage("Open the project in VS Code or run " + devStartScriptName() + " to start the workspace task.")
		m.AppendStepMessage("To start the server manually: cd " + m.ProjectName + " && " + devServerCommand())
		m.AppendStepMessage("Use " + superuserGuideFile + " for admin account setup and stick to the .venv-based commands documented there.")

		if m.SetupTailwind {
			m.AppendStepMessage("To watch Tailwind CSS: cd " + m.ProjectName + " && npm run watch:css")
		}
	}
}

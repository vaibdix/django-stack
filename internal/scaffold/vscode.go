package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// writeVSCodeConfig writes .vscode/ (tasks, launch, extensions, settings) and a
// WELCOME.md into the generated project.
//
// Two exported wrappers used to sit here — CreateVSCodeFiles, a one-line
// pass-through to this function, and OpenVSCode, which shelled out to `code`.
// Neither had a single caller anywhere in the repo.
func writeVSCodeConfig(projectPath string, setupTailwind bool) {
	vscodeDir := filepath.Join(projectPath, ".vscode")
	_ = os.MkdirAll(vscodeDir, 0755)

	presentation := map[string]any{
		"group":  "dev",
		"reveal": "always",
		"panel":  "dedicated",
		"focus":  false,
		"clear":  false,
	}

	tasks := []map[string]any{
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
		tasks[0]["options"] = map[string]any{
			"shell": map[string]any{
				"executable": "cmd.exe",
				"args":       []string{"/c"},
			},
		}
	}

	if setupTailwind {
		tasks = append(tasks, map[string]any{
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
	tasks = append(tasks, map[string]any{
		"label":        "Workspace: Start Dev",
		"dependsOn":    startTaskDependencies,
		"dependsOrder": "parallel",
		"runOptions": map[string]any{
			"runOn": "folderOpen",
		},
		"problemMatcher": []string{},
	})

	tasksObject := map[string]any{
		"version": "2.0.0",
		"tasks":   tasks,
	}
	tasksJSON, _ := json.MarshalIndent(tasksObject, "", "  ")
	_ = os.WriteFile(filepath.Join(vscodeDir, "tasks.json"), tasksJSON, 0644)

	launchConfigs := []map[string]any{
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

	compounds := []map[string]any{}
	if setupTailwind {
		launchConfigs = append(launchConfigs, map[string]any{
			"name":              "Tailwind: Watch CSS",
			"type":              "node",
			"request":           "launch",
			"runtimeExecutable": "npm",
			"runtimeArgs":       []string{"run", "watch:css"},
			"console":           "integratedTerminal",
		})
		compounds = append(compounds, map[string]any{
			"name":           "Django + Tailwind",
			"configurations": []string{"Django: Run Server", "Tailwind: Watch CSS"},
			"stopAll":        true,
		})
	}

	launchObject := map[string]any{
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

// reportServerInstructions logs how to start the project by hand.
func (e *Env) reportServerInstructions() {
	if e.Cfg.RunServer {
		e.Step("Open the project in VS Code or run " + devStartScriptName() + " to start the workspace task.")
		e.Step("To start the server manually: cd " + e.Cfg.ProjectName + " && " + devServerCommand())
		e.Step("Use " + superuserGuideFile + " for admin account setup and stick to the .venv-based commands documented there.")

		if e.Cfg.SetupTailwind {
			e.Step("To watch Tailwind CSS: cd " + e.Cfg.ProjectName + " && npm run watch:css")
		}
	}
}

package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type vscodeTask struct {
	Label        string              `json:"label"`
	Type         string              `json:"type,omitzero"`
	Command      string              `json:"command,omitzero"`
	Options      *vscodeTaskOptions  `json:"options,omitzero"`
	IsBackground bool                `json:"isBackground,omitzero"`
	Presentation *vscodePresentation `json:"presentation,omitzero"`
	DependsOn    []string            `json:"dependsOn,omitzero"`
	DependsOrder string              `json:"dependsOrder,omitzero"`
	RunOptions   *vscodeRunOptions   `json:"runOptions,omitzero"`

	ProblemMatcher []string `json:"problemMatcher"`
}

type vscodePresentation struct {
	Group  string `json:"group"`
	Reveal string `json:"reveal"`
	Panel  string `json:"panel"`
	Focus  bool   `json:"focus"`
	Clear  bool   `json:"clear"`
}

type vscodeTaskOptions struct {
	Shell vscodeShell `json:"shell"`
}

type vscodeShell struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
}

type vscodeRunOptions struct {
	RunOn string `json:"runOn"`
}

type vscodeTasksFile struct {
	Version string       `json:"version"`
	Tasks   []vscodeTask `json:"tasks"`
}

type vscodeLaunchConfig struct {
	Name              string   `json:"name"`
	Type              string   `json:"type"`
	Request           string   `json:"request"`
	Program           string   `json:"program,omitzero"`
	Args              []string `json:"args,omitzero"`
	Django            bool     `json:"django,omitzero"`
	JustMyCode        bool     `json:"justMyCode,omitzero"`
	RuntimeExecutable string   `json:"runtimeExecutable,omitzero"`
	RuntimeArgs       []string `json:"runtimeArgs,omitzero"`
	Console           string   `json:"console"`
}

type vscodeCompound struct {
	Name           string   `json:"name"`
	Configurations []string `json:"configurations"`
	StopAll        bool     `json:"stopAll"`
}

type vscodeLaunchFile struct {
	Version        string               `json:"version"`
	Configurations []vscodeLaunchConfig `json:"configurations"`
	Compounds      []vscodeCompound     `json:"compounds,omitzero"`
}

type workspace struct {
	Tailwind bool
	Windows  bool
}

const (
	djangoServerTask  = "Django: Run server"
	tailwindWatchTask = "Tailwind: Watch CSS"
	startDevTask      = "Workspace: Start Dev"
)

func (w workspace) tasksFile() vscodeTasksFile {
	presentation := &vscodePresentation{
		Group:  "dev",
		Reveal: "always",
		Panel:  "dedicated",
		Focus:  false,
		Clear:  false,
	}

	server := vscodeTask{
		Label:          djangoServerTask,
		Type:           "shell",
		Command:        devServerCommandFor(w.Windows),
		IsBackground:   true,
		Presentation:   presentation,
		ProblemMatcher: []string{},
	}
	if w.Windows {

		server.Options = &vscodeTaskOptions{
			Shell: vscodeShell{Executable: "cmd.exe", Args: []string{"/c"}},
		}
	}

	tasks := []vscodeTask{server}
	dependsOn := []string{djangoServerTask}

	if w.Tailwind {
		tasks = append(tasks, vscodeTask{
			Label:          tailwindWatchTask,
			Type:           "shell",
			Command:        "npm run watch:css",
			IsBackground:   true,
			Presentation:   presentation,
			ProblemMatcher: []string{},
		})
		dependsOn = append(dependsOn, tailwindWatchTask)
	}

	tasks = append(tasks, vscodeTask{
		Label:          startDevTask,
		DependsOn:      dependsOn,
		DependsOrder:   "parallel",
		RunOptions:     &vscodeRunOptions{RunOn: "folderOpen"},
		ProblemMatcher: []string{},
	})

	return vscodeTasksFile{Version: "2.0.0", Tasks: tasks}
}

func (w workspace) launchFile() vscodeLaunchFile {
	launch := vscodeLaunchFile{
		Version: "0.2.0",
		Configurations: []vscodeLaunchConfig{{
			Name:       "Django: Run Server",
			Type:       "debugpy",
			Request:    "launch",
			Program:    "${workspaceFolder}/manage.py",
			Args:       []string{"runserver", "--noreload"},
			Django:     true,
			JustMyCode: true,
			Console:    "integratedTerminal",
		}},
	}

	if w.Tailwind {
		launch.Configurations = append(launch.Configurations, vscodeLaunchConfig{
			Name:              tailwindWatchTask,
			Type:              "node",
			Request:           "launch",
			RuntimeExecutable: "npm",
			RuntimeArgs:       []string{"run", "watch:css"},
			Console:           "integratedTerminal",
		})
		launch.Compounds = []vscodeCompound{{
			Name:           "Django + Tailwind",
			Configurations: []string{"Django: Run Server", tailwindWatchTask},
			StopAll:        true,
		}}
	}

	return launch
}

func (w workspace) settingsJSON() string {
	if w.Windows {
		return `{
  "task.allowAutomaticTasks": "on",
  "terminal.integrated.defaultProfile.windows": "Command Prompt"
}`
	}
	return `{
  "task.allowAutomaticTasks": "on"
}`
}

const vscodeExtensionsJSON = `{
  "recommendations": [
    "ms-python.python",
    "esbenp.prettier-vscode",
    "bradlc.vscode-tailwindcss"
  ]
}`

func (w workspace) welcome(projectDirName string) string {
	return render("welcome.md.tmpl", struct {
		Tailwind        bool
		StartScriptName string
		SuperuserGuide  string
		ProjectDirName  string
	}{
		Tailwind:        w.Tailwind,
		StartScriptName: devStartScriptNameFor(w.Windows),
		SuperuserGuide:  superuserGuideFile,
		ProjectDirName:  projectDirName,
	})
}

func marshalVSCode(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, nil
}

func writeVSCodeConfig(projectPath string, setupTailwind bool) error {
	return workspace{
		Tailwind: setupTailwind,
		Windows:  runtime.GOOS == "windows",
	}.write(projectPath)
}

func (w workspace) write(projectPath string) error {
	vscodeDir := filepath.Join(projectPath, ".vscode")
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vscode directory: %w", err)
	}

	tasks, err := marshalVSCode(w.tasksFile())
	if err != nil {
		return fmt.Errorf("failed to encode tasks.json: %w", err)
	}
	launch, err := marshalVSCode(w.launchFile())
	if err != nil {
		return fmt.Errorf("failed to encode launch.json: %w", err)
	}

	files := map[string][]byte{
		filepath.Join(vscodeDir, "tasks.json"):      tasks,
		filepath.Join(vscodeDir, "launch.json"):     launch,
		filepath.Join(vscodeDir, "extensions.json"): []byte(vscodeExtensionsJSON),
		filepath.Join(vscodeDir, "settings.json"):   []byte(w.settingsJSON()),
		filepath.Join(projectPath, "WELCOME.md"):    []byte(w.welcome(filepath.Base(projectPath))),
	}
	for path, content := range files {
		if err := os.WriteFile(path, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath.Base(path), err)
		}
	}
	return nil
}

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

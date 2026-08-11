package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// The VS Code workspace written into a generated project: .vscode/ plus the
// WELCOME.md that explains it.
//
// These files used to be built from map[string]any literals — thirteen of them,
// nested three deep — and the WELCOME.md from twenty-three string appends. That
// is the same untyped shape this codebase removed from the page renderer and
// the feature flags, and it fails the same way: a mistyped key is not an error,
// it is a task VS Code silently ignores. The structs below are checked by the
// compiler and pinned by goldens.

// vscodeTask is one entry in tasks.json.
//
// Most fields are optional because the composite "Workspace: Start Dev" task
// carries none of them: it has no command of its own, it only depends on the
// others. omitzero rather than omitempty — the two agree on strings and bools,
// but omitempty would keep an all-zero Presentation struct, which is exactly
// the field that must disappear.
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

	// ProblemMatcher is always written, including when empty: VS Code prompts
	// the user to pick one if the key is absent, which defeats the point of a
	// task that is supposed to start on its own. It is therefore not omitzero,
	// and must be non-nil so it marshals as [] rather than null.
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

// vscodeLaunchConfig is one debug configuration in launch.json.
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

// vscodeCompound runs several configurations together. It belongs at the top
// level of launch.json, beside "configurations" rather than inside it.
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

// workspace is what the generated VS Code configuration depends on.
//
// Windows is a field rather than a read of runtime.GOOS at each branch, for the
// same reason startScriptData carries the platform: the goldens have to cover
// both platforms from whichever host runs the tests, and a package-level
// runtime.GOOS makes that impossible.
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
		// cmd.exe, not PowerShell: the command is a plain path invocation and
		// PowerShell will not run one from the current directory without a
		// leading .\, which the Unix form does not have.
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

// settingsJSON enables the folderOpen task. Without it VS Code asks for
// permission on every open, which is the one interaction the auto-start task
// exists to avoid.
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

// marshalVSCode renders a workspace file the way VS Code's own generated files
// are formatted: two-space indent, trailing newline.
func marshalVSCode(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, nil
}

// writeVSCodeConfig writes .vscode/ (tasks, launch, extensions, settings) and a
// WELCOME.md into the generated project.
//
// Two exported wrappers used to sit here — CreateVSCodeFiles, a one-line
// pass-through to this function, and OpenVSCode, which shelled out to `code`.
// Neither had a single caller anywhere in the repo.
func writeVSCodeConfig(projectPath string, setupTailwind bool) error {
	return workspace{
		Tailwind: setupTailwind,
		Windows:  runtime.GOOS == "windows",
	}.write(projectPath)
}

// write emits every workspace file into projectPath.
//
// It returns an error where the previous version discarded all five. A silently
// missing tasks.json is a project that does not start its dev server on open —
// the entire feature this file exists for — reported to the user as success.
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

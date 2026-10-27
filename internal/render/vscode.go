package render

import (
	"encoding/json"
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

type vscodeTask struct {
	Label        string              `json:"label"`
	Type         string              `json:"type,omitzero"`
	Command      string              `json:"command,omitzero"`
	Options      *vscodeTaskOptions  `json:"options,omitzero"`
	Windows      *vscodePlatformTask `json:"windows,omitzero"`
	IsBackground bool                `json:"isBackground,omitzero"`
	Presentation *vscodePresentation `json:"presentation,omitzero"`
	DependsOn    []string            `json:"dependsOn,omitzero"`
	DependsOrder string              `json:"dependsOrder,omitzero"`
	RunOptions   *vscodeRunOptions   `json:"runOptions,omitzero"`

	ProblemMatcher []string `json:"problemMatcher"`
}

type vscodePlatformTask struct {
	Command string             `json:"command"`
	Options *vscodeTaskOptions `json:"options,omitzero"`
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

type Workspace struct {
	Tailwind    bool
	NodeManager NodeManager
	Project     config.Spec
}

const (
	djangoServerTask  = "Django: Run server"
	tailwindWatchTask = "Tailwind: Watch CSS"
	startDevTask      = "Workspace: Start Dev"
)

func (w Workspace) tasksFile() vscodeTasksFile {
	presentation := &vscodePresentation{
		Group:  "dev",
		Reveal: "always",
		Panel:  "dedicated",
		Focus:  false,
		Clear:  false,
	}

	server := vscodeTask{
		Label:   djangoServerTask,
		Type:    "shell",
		Command: devServerCommandFor("linux"),
		Windows: &vscodePlatformTask{
			Command: devServerCommandFor(pyenv.Windows),
			Options: &vscodeTaskOptions{Shell: vscodeShell{Executable: "cmd.exe", Args: []string{"/d", "/c"}}},
		},
		IsBackground:   true,
		Presentation:   presentation,
		ProblemMatcher: []string{},
	}

	tasks := []vscodeTask{server}
	dependsOn := []string{djangoServerTask}

	if w.Tailwind {
		tasks = append(tasks, vscodeTask{
			Label:          tailwindWatchTask,
			Type:           "shell",
			Command:        w.NodeManager.RunCommand("watch:css"),
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

func (w Workspace) launchFile() vscodeLaunchFile {
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
			RuntimeExecutable: w.NodeManager.Name,
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

type vscodeExtensions struct {
	Recommendations []string `json:"recommendations"`
}

func extensionsFile(cfg config.Spec) vscodeExtensions {
	recommendations := []string{"ms-python.python", "batisteo.vscode-django"}
	if cfg.Has(config.Ruff) {
		recommendations = append(recommendations, "charliermarsh.ruff")
	}
	if cfg.Has(config.Mypy) {
		recommendations = append(recommendations, "ms-python.mypy-type-checker")
	}
	if cfg.CreateTemplates() {
		recommendations = append(recommendations, "monosans.djlint")
	}
	if cfg.UsesTailwind() {
		recommendations = append(recommendations, "bradlc.vscode-tailwindcss")
	}
	return vscodeExtensions{Recommendations: recommendations}
}

func (w Workspace) welcome() string {
	return Render("welcome.md.tmpl", struct {
		Tailwind   bool
		Templates  bool
		ModuleName string
	}{
		Tailwind:   w.Tailwind,
		Templates:  w.Project.CreateTemplates(),
		ModuleName: w.Project.ModuleName(),
	})
}

func marshalVSCode(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func (p Project) Workspace() Workspace {
	return Workspace{
		Tailwind:    p.Cfg.UsesTailwind(),
		NodeManager: p.Node,
		Project:     p.Cfg,
	}
}

func (p Project) workspaceFiles() (FileSet, error) {
	return p.Workspace().Files()
}

func (w Workspace) Files() (FileSet, error) {
	tasks, err := marshalVSCode(w.tasksFile())
	if err != nil {
		return nil, fmt.Errorf("encode tasks.json: %w", err)
	}
	launch, err := marshalVSCode(w.launchFile())
	if err != nil {
		return nil, fmt.Errorf("encode launch.json: %w", err)
	}
	extensions, err := marshalVSCode(extensionsFile(w.Project))
	if err != nil {
		return nil, fmt.Errorf("encode extensions.json: %w", err)
	}

	files := FileSet{}
	files.addAll(".vscode", map[string]string{
		"tasks.json":      string(tasks),
		"launch.json":     string(launch),
		"extensions.json": string(extensions),
	})
	files.add("WELCOME.md", w.welcome(), FileMode)
	return files, nil
}

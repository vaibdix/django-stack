package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/golden"

	"github.com/vaibdix/django-stack/internal/config"
)

func workspaceVariants() map[string]workspace {
	return map[string]workspace{
		"plain":         {NodeManager: NPM},
		"tailwind":      {Tailwind: true, NodeManager: NPM},
		"tailwind_pnpm": {Tailwind: true, NodeManager: PNPM},
	}
}

func renderWorkspace(t *testing.T, w workspace) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "my-shop")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("creating project dir: %v", err)
	}
	if err := w.write(root); err != nil {
		t.Fatalf("writing workspace: %v", err)
	}

	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking generated workspace: %v", err)
	}
	slices.Sort(paths)

	var b strings.Builder
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatalf("relativising %s: %v", path, err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		b.WriteString("=== " + filepath.ToSlash(rel) + " ===\n")
		b.Write(content)
		b.WriteString("\n")
	}
	return b.String()
}

func TestGoldenVSCodeWorkspace(t *testing.T) {
	t.Parallel()

	for name, w := range workspaceVariants() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			golden.Assert(t, "vscode/"+name+".txt", renderWorkspace(t, w))
		})
	}
}

func TestGeneratedWorkspaceJSONIsValid(t *testing.T) {
	t.Parallel()

	for name, w := range workspaceVariants() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			if err := w.write(root); err != nil {
				t.Fatalf("writing workspace: %v", err)
			}

			for _, file := range []string{"tasks.json", "launch.json", "extensions.json"} {
				raw, err := os.ReadFile(filepath.Join(root, ".vscode", file))
				if err != nil {
					t.Fatalf("reading %s: %v", file, err)
				}
				var parsed any
				if err := json.Unmarshal(raw, &parsed); err != nil {
					t.Errorf("%s is not valid JSON: %v\n%s", file, err, raw)
				}
			}
		})
	}
}

func TestWorkspaceTasksCarryTheirLoadBearingFields(t *testing.T) {
	t.Parallel()

	for name, w := range workspaceVariants() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tasks := w.tasksFile()

			byLabel := make(map[string]vscodeTask, len(tasks.Tasks))
			for _, task := range tasks.Tasks {
				byLabel[task.Label] = task
			}

			start, ok := byLabel[startDevTask]
			if !ok {
				t.Fatalf("no %q task; nothing runs when the folder is opened", startDevTask)
			}
			if start.RunOptions == nil || start.RunOptions.RunOn != "folderOpen" {
				t.Errorf("%q does not run on folderOpen, so the workspace starts nothing automatically", startDevTask)
			}
			if start.DependsOrder != "parallel" {
				t.Errorf("dependsOrder = %q, want parallel: the server and the CSS watcher must not serialise", start.DependsOrder)
			}
			if start.Command != "" || start.Presentation != nil || start.IsBackground {
				t.Errorf("the composite task carries command/presentation/isBackground fields it should omit: %+v", start)
			}

			server, ok := byLabel[djangoServerTask]
			if !ok {
				t.Fatalf("no %q task", djangoServerTask)
			}
			if !server.IsBackground {
				t.Error("the server task is not marked isBackground, so VS Code waits for it to exit")
			}
			if !strings.Contains(server.Command, ".venv") {
				t.Errorf("server command %q does not use the project venv", server.Command)
			}

			for _, task := range tasks.Tasks {
				if task.ProblemMatcher == nil {
					t.Errorf("task %q has a nil problemMatcher; it marshals as null and VS Code prompts", task.Label)
				}
			}

			if _, hasTailwind := byLabel[tailwindWatchTask]; hasTailwind != w.Tailwind {
				t.Errorf("Tailwind=%v but the watch task present=%v", w.Tailwind, hasTailwind)
			}
			if !slices.Contains(start.DependsOn, djangoServerTask) {
				t.Errorf("%q does not depend on the server task: %v", startDevTask, start.DependsOn)
			}
			if w.Tailwind && !slices.Contains(start.DependsOn, tailwindWatchTask) {
				t.Errorf("Tailwind is on but %q does not depend on the watch task: %v", startDevTask, start.DependsOn)
			}

			if server.Windows == nil || server.Windows.Options == nil || !strings.Contains(server.Windows.Command, `\Scripts\`) {
				t.Errorf("the server task has no Windows override with cmd.exe and the Scripts interpreter: %+v", server.Windows)
			}
			if server.Options != nil || strings.Contains(server.Command, `\`) {
				t.Errorf("Windows details leaked into the default (macOS/Linux) command: %q", server.Command)
			}
		})
	}
}

func TestWriteVSCodeConfigWiresItsArgumentsThrough(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		t.Run(fmt.Sprintf("tailwind=%v", tailwind), func(t *testing.T) {
			t.Parallel()

			projectPath := t.TempDir()
			if err := writeVSCodeConfig(projectPath, config.Project{SetupTailwind: tailwind}, NPM); err != nil {
				t.Fatalf("writeVSCodeConfig: %v", err)
			}

			for _, rel := range []string{
				".vscode/tasks.json", ".vscode/launch.json",
				".vscode/extensions.json", "WELCOME.md",
			} {
				if _, err := os.Stat(filepath.Join(projectPath, filepath.FromSlash(rel))); err != nil {
					t.Errorf("%s was not written: %v", rel, err)
				}
			}

			tasks, err := os.ReadFile(filepath.Join(projectPath, ".vscode", "tasks.json"))
			if err != nil {
				t.Fatalf("reading tasks.json: %v", err)
			}
			if got := strings.Contains(string(tasks), tailwindWatchTask); got != tailwind {
				t.Errorf("called with tailwind=%v but the watch task present=%v; "+
					"the flag is not reaching the generated workspace", tailwind, got)
			}
		})
	}
}

func TestLaunchCompoundsSitAtTheTopLevel(t *testing.T) {
	t.Parallel()

	withTailwind := workspace{Tailwind: true, NodeManager: NPM}.launchFile()
	if len(withTailwind.Compounds) != 1 {
		t.Fatalf("Tailwind is on but launch.json declares %d compounds, want 1", len(withTailwind.Compounds))
	}
	if len(withTailwind.Configurations) != 2 {
		t.Errorf("want a Django and a Tailwind configuration, got %d", len(withTailwind.Configurations))
	}

	raw, err := marshalVSCode(workspace{NodeManager: NPM}.launchFile())
	if err != nil {
		t.Fatalf("encoding launch.json: %v", err)
	}
	if strings.Contains(string(raw), "compounds") {
		t.Errorf("Tailwind is off but launch.json still declares compounds:\n%s", raw)
	}
}

func TestRecommendedExtensionsMatchTheToolsTheProjectActuallyUses(t *testing.T) {
	t.Parallel()

	full := vscodeExtensionsJSON(config.Project{
		SetupRuff: true, SetupMypy: true, SetupTailwind: true, CreateTemplates: true,
	})
	for _, want := range []string{"charliermarsh.ruff", "ms-python.mypy-type-checker", "monosans.djlint", "bradlc.vscode-tailwindcss"} {
		if !strings.Contains(full, want) {
			t.Errorf("a project that uses that tool does not recommend %s:\n%s", want, full)
		}
	}

	bare := vscodeExtensionsJSON(config.Project{})
	for _, unwanted := range []string{"tailwindcss", "ruff", "mypy", "djlint", "prettier"} {
		if strings.Contains(bare, unwanted) {
			t.Errorf("a project without that tool still recommends %s:\n%s", unwanted, bare)
		}
	}
	if !strings.Contains(bare, "ms-python.python") {
		t.Errorf("every Django project wants the Python extension:\n%s", bare)
	}
}

func TestTheWorkspaceNeitherForcesAutomaticTasksNorDependsOnTheGeneratingMachine(t *testing.T) {
	t.Parallel()

	for name, w := range workspaceVariants() {
		files := renderWorkspace(t, w)
		if strings.Contains(files, "allowAutomaticTasks") {
			t.Errorf("%s: the workspace overrides VS Code's automatic-task consent:\n%s", name, files)
		}
		if strings.Contains(files, "settings.json") {
			t.Errorf("%s: a workspace settings.json is still written", name)
		}
	}

	for _, windows := range []bool{false, true} {
		script := generateStartScript(windows, true)
		for _, machinePath := range []string{"/usr/", "/home/", `C:\`, "Program Files"} {
			if strings.Contains(script, machinePath) {
				t.Errorf("windows=%v: the start script embeds a machine path %q:\n%s", windows, machinePath, script)
			}
		}
	}
}

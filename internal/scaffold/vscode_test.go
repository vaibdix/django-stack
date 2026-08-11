package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Golden coverage for the VS Code workspace.
//
// These files had none, which is why the map[string]any version could not be
// refactored with any confidence: nothing said what it was supposed to emit.
// One golden per configuration holds every generated file, so a review reads
// the whole workspace in one diff rather than five.
//
// The platform is a field on workspace rather than a read of runtime.GOOS,
// so both platforms are covered from whichever host runs the tests. Without
// that, half of these goldens could only ever be produced on Windows.

func workspaceVariants() map[string]workspace {
	return map[string]workspace{
		"plain_unix":       {},
		"tailwind_unix":    {Tailwind: true},
		"plain_windows":    {Windows: true},
		"tailwind_windows": {Tailwind: true, Windows: true},
	}
}

// renderWorkspace writes a workspace to a temp dir and reads every file back as
// one document, so the golden records the file names as well as the contents.
func renderWorkspace(t *testing.T, w workspace) string {
	t.Helper()

	// A fixed directory name: the project's own name reaches WELCOME.md, and a
	// t.TempDir() path would put a different random suffix in every golden.
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
			assertGolden(t, "vscode/"+name+".txt", renderWorkspace(t, w))
		})
	}
}

// The generated JSON has to parse. A golden pins bytes and would happily record
// a trailing comma or a doubled brace forever.
func TestGeneratedWorkspaceJSONIsValid(t *testing.T) {
	t.Parallel()

	for name, w := range workspaceVariants() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			if err := w.write(root); err != nil {
				t.Fatalf("writing workspace: %v", err)
			}

			for _, file := range []string{"tasks.json", "launch.json", "settings.json", "extensions.json"} {
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

// The properties that make the workspace actually start something. Each one is
// a silent failure if it goes missing: VS Code ignores what it does not
// understand, so a wrong key produces a workspace that opens and does nothing.
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

			// problemMatcher must be present-and-empty on every task, or VS Code
			// prompts for one and the auto-start never happens unattended.
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

			// The Windows shell override exists because the command is a bare
			// relative path, which PowerShell refuses to run.
			if w.Windows && server.Options == nil {
				t.Error("no cmd.exe shell override on Windows; PowerShell will not run the relative venv path")
			}
			if !w.Windows && server.Options != nil {
				t.Error("a Windows shell override leaked into the Unix workspace")
			}
		})
	}
}

// The goldens and the contract tests above all drive workspace.write directly.
// This is the one test of the exported entry point: that it writes to the paths
// the scaffolder promises and passes its arguments through rather than
// defaulting them.
func TestWriteVSCodeConfigWiresItsArgumentsThrough(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		t.Run(fmt.Sprintf("tailwind=%v", tailwind), func(t *testing.T) {
			t.Parallel()

			projectPath := t.TempDir()
			if err := writeVSCodeConfig(projectPath, tailwind); err != nil {
				t.Fatalf("writeVSCodeConfig: %v", err)
			}

			for _, rel := range []string{
				".vscode/tasks.json", ".vscode/launch.json",
				".vscode/settings.json", ".vscode/extensions.json", "WELCOME.md",
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

// Compounds belong beside "configurations", not inside it — nesting them is
// accepted by the JSON parser and silently ignored by VS Code.
func TestLaunchCompoundsSitAtTheTopLevel(t *testing.T) {
	t.Parallel()

	withTailwind := workspace{Tailwind: true}.launchFile()
	if len(withTailwind.Compounds) != 1 {
		t.Fatalf("Tailwind is on but launch.json declares %d compounds, want 1", len(withTailwind.Compounds))
	}
	if len(withTailwind.Configurations) != 2 {
		t.Errorf("want a Django and a Tailwind configuration, got %d", len(withTailwind.Configurations))
	}

	raw, err := marshalVSCode(workspace{}.launchFile())
	if err != nil {
		t.Fatalf("encoding launch.json: %v", err)
	}
	if strings.Contains(string(raw), "compounds") {
		t.Errorf("Tailwind is off but launch.json still declares compounds:\n%s", raw)
	}
}

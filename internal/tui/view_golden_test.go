package tui

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files instead of comparing against them")

var goldenSizes = []struct {
	name          string
	width, height int
}{
	{name: "wide", width: 100, height: 32},
	{name: "narrow", width: 56, height: 28},
}

func TestViewGolden(t *testing.T) {

	if runtime.GOOS == "windows" {
		t.Skip("goldens record POSIX interpreter paths; see the comment above")
	}

	restore := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(restore) })

	for _, screen := range goldenScreens() {
		for _, size := range goldenSizes {
			t.Run(screen.name+"_"+size.name, func(t *testing.T) {
				m := screen.build()
				m.Width, m.Height = size.width, size.height
				m.syncFormWidths()
				if form := m.GetActiveForm(); form != nil {
					form.Init()
				}

				assertGolden(t, fmt.Sprintf("view_%s_%s.txt", screen.name, size.name), normalise(m.View()))
			})
		}
	}
}

type goldenScreen struct {
	name  string
	build func() *Model
}

func goldenScreens() []goldenScreen {
	base := func(step Step) *Model {
		m := NewModel()
		m.Step = step
		m.ProjectName = goldenProjectName
		m.DjangoVersion = "5.2.1"
		m.EyesOpen = true
		return m
	}

	return []goldenScreen{
		{"splash", func() *Model {
			m := base(StepSplashScreen)
			m.DjangoVersion = ""
			m.SplashCountdown = 3
			return m
		}},
		{"project_name", func() *Model { return base(StepProjectName) }},
		{"python", func() *Model {
			m := base(StepPythonSelection)

			m.PythonForm = buildPendingPythonForm(&m.SelectedPythonPath)
			return m
		}},
		{"django_version", func() *Model {
			m := base(StepDjangoVersion)
			m.DjangoVersionNote = "Showing the 5 latest compatible stable Django releases from PyPI."
			m.AvailableDjangoOptions = []DjangoOption{
				{Version: "5.2.1", Label: "5.2.1 (latest)"},
				{Version: "5.1.9", Label: "5.1.9"},
			}
			m.DjangoVersionForm = m.buildDjangoVersionForm()
			return m
		}},
		{"database", func() *Model { return base(StepDatabase) }},
		{"features", func() *Model { return base(StepProjectConfig) }},
		{"app_name", func() *Model { return base(StepAppName) }},
		{"setup", func() *Model {
			m := base(StepSetup)
			m.ProgressStatus = "Installing Django and project dependencies"
			m.CompletedSteps, m.TotalSteps = 9, 21
			m.StepMessages = []string{
				"Created the virtual environment",
				"Installed Django 5.2.1",
				"Configured the settings module",
			}
			return m
		}},
		{"dev_server_prompt", func() *Model { return base(StepDevServerPrompt) }},
		{"complete", func() *Model {
			m := base(StepComplete)
			m.SetupTailwind = true
			return m
		}},
		{"done_vscode", func() *Model {
			m := base(StepDevServerPrompt)
			m.Done, m.StartDevServer = true, true
			return m
		}},
		{"done_manual", func() *Model {
			m := base(StepDevServerPrompt)
			m.Done = true
			return m
		}},
		{"error", func() *Model {
			m := base(StepSetup)
			m.Error = fmt.Errorf("install packages: %w", os.ErrPermission)
			return m
		}},
	}
}

const goldenProjectName = "/projects/myshop"

func normalise(view string) string {
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	view = strings.Join(lines, "\n")

	m := &Model{}
	m.ProjectName = goldenProjectName
	return strings.ReplaceAll(view, m.venvPython(), "<venv-python>")
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", "golden", name)

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating golden dir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("writing golden %s: %v", name, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden %s: %v\n\nRun: go test ./internal/tui -update", name, err)
	}
	if got != string(want) {
		t.Errorf("rendered %s does not match its golden file.\n"+
			"If intentional, run: go test ./internal/tui -update\n\ngot:\n%s", name, got)
	}
}

func TestGoldenFormsActuallyRender(t *testing.T) {
	m := NewModel()
	m.Step = StepDatabase
	m.syncFormWidths()

	form := m.GetActiveForm()
	form.Init()

	if !strings.Contains(form.View(), "Choose Database") {
		t.Fatalf("the database form rendered nothing recognisable, so the form goldens are vacuous:\n%q", form.View())
	}
}

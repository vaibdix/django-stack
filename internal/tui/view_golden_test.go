package tui

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/golden"
	"github.com/vaibdix/django-stack/internal/releases"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

var goldenSizes = []struct {
	name          string
	width, height int
}{
	{name: "wide", width: 100, height: 32},
	{name: "narrow", width: 56, height: 28},
}

func TestViewGolden(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the goldens record POSIX interpreter paths")
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

				golden.Assert(t, fmt.Sprintf("view_%s_%s.txt", screen.name, size.name), normalise(m.View()))
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
		m := NewModel(scaffold.Options{})
		m.Step = step
		m.ProjectName = goldenProjectName
		m.DjangoVersion = "5.2.1"
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
			m.DjangoVersionNote = "The newest release of each Django series that supports Python 3.13, from PyPI."
			m.AvailableDjangoOptions = []releases.Option{
				{Version: "6.1.2", Label: "6.1.2 (latest)"},
				{Version: "6.0.9", Label: "6.0.9"},
				{Version: "5.2.18", Label: "5.2.18 (LTS)"},
			}
			m.DjangoVersionForm = m.buildDjangoVersionForm()
			return m
		}},
		{"database", func() *Model { return base(StepDatabase) }},
		{"docker_checking", func() *Model {
			m := base(StepDockerCheck)
			m.DatabaseType = config.PostgresDocker
			m.DockerChecking = true
			return m
		}},
		{"docker_stopped", func() *Model {
			m := base(StepDockerCheck)
			m.DatabaseType = config.PostgresDocker
			m.DockerProblem = dockerStoppedProblem
			m.DockerRemedy = DockerRetry
			m.DockerForm = buildDockerRemedyForm(&m.DockerRemedy, m.DatabaseType)
			return m
		}},
		{"stack", func() *Model { return base(StepStack) }},
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
			m.NodeManager = scaffold.NPM
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

func TestGoldenFormsActuallyRender(t *testing.T) {
	m := NewModel(scaffold.Options{})
	m.Step = StepDatabase
	m.syncFormWidths()

	form := m.GetActiveForm()
	form.Init()

	if !strings.Contains(form.View(), "Choose Database") {
		t.Fatalf("the database form rendered nothing recognisable, so the form goldens are vacuous:\n%q", form.View())
	}
}

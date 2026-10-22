package tui

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/golden"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/releases"

	"github.com/charmbracelet/colorprofile"

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
	if pyenv.IsWindows(runtime.GOOS) {
		t.Skip("the goldens record POSIX interpreter paths")
	}

	for _, screen := range goldenScreens() {
		for _, size := range goldenSizes {
			t.Run(screen.name+"_"+size.name, func(t *testing.T) {
				m := screen.build()
				m.width, m.height = size.width, size.height
				m.syncFormWidths()
				if form := m.currentForm(); form != nil {
					form.Init()
				}

				golden.Assert(t, fmt.Sprintf("view_%s_%s.txt", screen.name, size.name), normalise(plain(m.render())))
			})
		}
	}
}

type goldenScreen struct {
	name  string
	build func() *Model
}

func goldenScreens() []goldenScreen {
	base := func(step wizardStep) *Model {
		m := NewModel(scaffold.Options{})
		m.step = step
		m.project.ProjectName = goldenProjectName
		m.project.DjangoVersion = "5.2.1"
		return m
	}

	return []goldenScreen{
		{"splash", func() *Model {
			m := base(stepSplashScreen)
			m.project.DjangoVersion = ""
			m.splashCountdown = 3
			return m
		}},
		{"project_name", func() *Model { return base(stepProjectName) }},
		{"python", func() *Model {
			m := base(stepPythonSelection)

			m.pythonForm = buildPendingPythonForm(&m.project.SelectedPythonPath)
			return m
		}},
		{"django_version", func() *Model {
			m := base(stepDjangoVersion)
			m.djangoVersionNote = "The newest release of each Django series that supports Python 3.13, from PyPI."
			m.availableDjangoOptions = []releases.Option{
				{Version: "6.1.2", Label: "6.1.2 (latest)"},
				{Version: "6.0.9", Label: "6.0.9"},
				{Version: "5.2.18", Label: "5.2.18 (LTS)"},
			}
			m.djangoVersionForm = m.buildDjangoVersionForm()
			return m
		}},
		{"database", func() *Model { return base(stepDatabase) }},
		{"docker_checking", func() *Model {
			m := base(stepDockerCheck)
			m.project.DatabaseType = config.PostgresDocker
			m.dockerChecking = true
			return m
		}},
		{"docker_stopped", func() *Model {
			m := base(stepDockerCheck)
			m.project.DatabaseType = config.PostgresDocker
			m.dockerProblem = dockerStoppedProblem
			m.dockerRemedy = dockerRetry
			m.dockerForm = buildDockerRemedyForm(&m.dockerRemedy, m.project.DatabaseType)
			return m
		}},
		{"stack", func() *Model { return base(stepStack) }},
		{"features", func() *Model { return base(stepProjectConfig) }},
		{"app_name", func() *Model { return base(stepAppName) }},
		{"setup", func() *Model {
			m := base(stepSetup)
			m.progressStatus = "Installing Django and project dependencies"
			m.completedSteps, m.totalSteps = 9, 21
			m.stepMessages = []string{
				"Created the virtual environment",
				"Installed Django 5.2.1",
				"Configured the settings module",
			}
			return m
		}},
		{"dev_server_prompt", func() *Model { return base(stepDevServerPrompt) }},
		{"complete", func() *Model {
			m := base(stepComplete)
			m.project.SetupTailwind = true
			m.nodeManager = scaffold.NPM
			return m
		}},
		{"done_vscode", func() *Model {
			m := base(stepDevServerPrompt)
			m.done, m.startDevServer = true, true
			return m
		}},
		{"done_manual", func() *Model {
			m := base(stepDevServerPrompt)
			m.done = true
			return m
		}},
		{"error", func() *Model {
			m := base(stepSetup)
			m.failure = fmt.Errorf("install packages: %w", os.ErrPermission)
			return m
		}},
	}
}

const goldenProjectName = "/projects/myshop"

func plain(view string) string {
	var b strings.Builder
	w := &colorprofile.Writer{Forward: &b, Profile: colorprofile.NoTTY}
	_, _ = w.WriteString(view)
	return b.String()
}

func normalise(view string) string {
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	view = strings.Join(lines, "\n")

	m := &Model{}
	m.project.ProjectName = goldenProjectName
	return strings.ReplaceAll(view, m.venvPython(), "<venv-python>")
}

func TestGoldenFormsActuallyRender(t *testing.T) {
	m := NewModel(scaffold.Options{})
	m.step = stepDatabase
	m.syncFormWidths()

	form := m.currentForm()
	form.Init()

	if !strings.Contains(form.View(), "Choose Database") {
		t.Fatalf("the database form rendered nothing recognisable, so the form goldens are vacuous:\n%q", form.View())
	}
}

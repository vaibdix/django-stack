package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

func NewModel() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f0b8ec")).
		Bold(true).
		MarginRight(2)

	p := progress.New(
		progress.WithGradient("#baa4ed", "#ffbc9e"),
		progress.WithWidth(50),
		progress.WithFillCharacters('●', '○'),
	)
	p.ShowPercentage = true

	m := &Model{
		ProjectConfig: ProjectConfig{
			ProjectName:        "django-stack",
			CreateTemplates:    false,
			CreateAppTemplates: false,
			RunServer:          false,
			InitializeGit:      false,
			DatabaseType:       "sqlite",
		},
		Spinner:         s,
		Progress:        p,
		Step:            StepSplashScreen,
		SplashCountdown: 3,
		ProgressStatus:  "Initializing...",
		SelectedOptions: []string{},
		CompletedSteps:  0,
		BlinkTimer:      time.Now(),
		EyesOpen:        true,
	}

	m.ProjectNameForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project Name").
				Description("This becomes the project folder name. Hyphens are allowed; Django module imports use underscores automatically.").
				Value(&m.ProjectName),
		),
	)

	m.RefreshPythonOptions()
	m.RefreshDjangoOptions()

	m.DatabaseForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose Database").
				Options(
					huh.NewOption("SQLite (default)", "sqlite"),
					huh.NewOption("PostgreSQL (Docker)", "postgres-docker"),
					huh.NewOption("MySQL (Docker)", "mysql-docker"),
				).
				Value(&m.DatabaseType),
		),
	)

	m.ProjectConfigForm = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Project Configuration"+"\n").
				Options(
					huh.NewOption("Standard Django Project (includes templates & static files)", "Standard Django Project"),
					huh.NewOption("Initialize Git Repository", "Initialize Git"),
					huh.NewOption("Tailwind CSS v4", "Tailwind"),
					huh.NewOption("Django REST Framework API", "REST Framework"),
					huh.NewOption("HTMX (modern interactions without JS)", "HTMX"),
				huh.NewOption("Django AllAuth (authentication)", "AllAuth"),
					huh.NewOption("Ruff (Linter + Formatter)", "Ruff"),
					huh.NewOption("Mypy (Type Checking)", "Mypy"),
					huh.NewOption("Pre-commit Hooks (requires Git)", "Pre-commit"),
					huh.NewOption("Multi-env Settings (dev/staging/prod)", "Multi-env"),
				).
				Limit(10).
				Value(&m.SelectedOptions),
		),
	)

	m.AppNameForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter app name").
				Value(&m.AppName),
		),
	)

	m.DevServerForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Open and run in VS Code?").
				Affirmative("Yes").
				Negative("No").
				Value(&m.StartDevServer),
		),
	)

	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
			return TickMsg{}
		}),
		m.Spinner.Tick,
	)
}

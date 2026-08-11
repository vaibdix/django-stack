package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/vaibdix/django-cli/internal/config"
)

// NewModel returns a Model ready for tea.NewProgram.
//
// It performs no I/O: interpreter discovery and the PyPI release lookup are
// commands issued by Init, so the model can be built in a test with no Python
// installed and no network.
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
		Project: config.Project{
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
		SelectedOptions: []config.Feature{},
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

	// The pickers start in a pending state and are replaced once Init's
	// discovery commands report back. NewModel does no I/O at all, so the model
	// can be constructed in a test without a Python installation or a network.
	m.PythonForm = buildPendingPythonForm(&m.SelectedPythonPath)
	m.DjangoVersionForm = buildPendingDjangoForm(&m.DjangoVersion)

	m.DatabaseForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[config.Database]().
				Title("Choose Database").
				Options(
					huh.NewOption("SQLite (default)", config.SQLite),
					huh.NewOption("PostgreSQL (Docker)", config.PostgresDocker),
					huh.NewOption("MySQL (Docker)", config.MySQLDocker),
				).
				Value(&m.DatabaseType),
		),
	)

	m.ProjectConfigForm = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[config.Feature]().
				Title("Project Configuration"+"\n").
				Options(
					huh.NewOption("Standard Django Project (includes templates & static files)", config.FeatureStandardProject),
					huh.NewOption("Initialize Git Repository", config.FeatureGit),
					huh.NewOption("Tailwind CSS v4", config.FeatureTailwind),
					huh.NewOption("Django REST Framework API", config.FeatureRestFramework),
					huh.NewOption("HTMX (modern interactions without JS)", config.FeatureHTMX),
					huh.NewOption("Django AllAuth (authentication)", config.FeatureAllAuth),
					huh.NewOption("Ruff (Linter + Formatter)", config.FeatureRuff),
					huh.NewOption("Mypy (Type Checking)", config.FeatureMypy),
					huh.NewOption("Pre-commit Hooks (requires Git)", config.FeaturePreCommit),
					huh.NewOption("Multi-env Settings (dev/staging/prod)", config.FeatureMultiEnv),
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

// Init starts the splash countdown and kicks off interpreter discovery.
//
// Discovery is a command rather than part of NewModel so that the splash screen
// renders immediately: the three-second countdown and the interpreter probe now
// overlap instead of running back to back.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
			return TickMsg{}
		}),
		m.Spinner.Tick,
		discoverPythonCmd(),
	)
}

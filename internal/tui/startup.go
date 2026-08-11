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

func NewModel() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f0b8ec")).
		Bold(true).
		MarginRight(2)

	p := progress.New(
		progress.WithGradient("#aa3bff", "#f0a3ff"),
		progress.WithWidth(50),
		progress.WithFillCharacters('●', '○'),
	)
	p.ShowPercentage = false

	m := &Model{
		Project: config.Project{
			ProjectName:        "django-stack",
			CreateTemplates:    false,
			CreateAppTemplates: false,
			RunServer:          false,
			InitializeGit:      false,
			DatabaseType:       "sqlite",
		},
		SelectedStack:   config.StackStandard,
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

	m.ProjectNameForm = buildProjectNameForm(&m.ProjectName)
	m.PythonForm = buildPendingPythonForm(&m.SelectedPythonPath)
	m.DjangoVersionForm = buildPendingDjangoForm(&m.DjangoVersion)
	m.DatabaseForm = buildDatabaseForm(&m.DatabaseType)
	m.StackForm = buildStackForm(&m.SelectedStack)
	m.ProjectConfigForm = buildProjectConfigForm(&m.SelectedOptions)
	m.AppNameForm = buildAppNameForm(&m.AppName)
	m.DevServerForm = buildDevServerForm(&m.StartDevServer)

	return m
}

func buildProjectNameForm(target *string) *huh.Form {
	return newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project Name").
				Description("This becomes the project folder name. Hyphens are allowed; Django module imports use underscores automatically.").
				Validate(validateProjectName).
				Value(target),
		),
	)
}

func validateProjectName(name string) error {
	return config.Project{ProjectName: name}.ValidateProjectName()
}

func buildDatabaseForm(target *config.Database) *huh.Form {
	return newForm(
		huh.NewGroup(
			huh.NewSelect[config.Database]().
				Title("Choose Database").
				Options(
					huh.NewOption("SQLite (default)", config.SQLite),
					huh.NewOption("PostgreSQL (Docker)", config.PostgresDocker),
					huh.NewOption("MySQL (Docker)", config.MySQLDocker),
				).
				Value(target),
		),
	)
}

func buildProjectConfigForm(target *[]config.Feature) *huh.Form {
	return newForm(
		huh.NewGroup(
			huh.NewMultiSelect[config.Feature]().
				Title("Project Configuration"+"\n").
				Options(
					huh.NewOption("Initialize Git Repository", config.FeatureGit),
					huh.NewOption("Django REST Framework API", config.FeatureRestFramework),
					huh.NewOption("Django AllAuth (authentication)", config.FeatureAllAuth),
					huh.NewOption("Ruff (Linter + Formatter)", config.FeatureRuff),
					huh.NewOption("Mypy (Type Checking)", config.FeatureMypy),
					huh.NewOption("Pre-commit Hooks (requires Git)", config.FeaturePreCommit),
					huh.NewOption("Multi-env Settings (dev/staging/prod)", config.FeatureMultiEnv),
				).
				Limit(7).
				Value(target),
		),
	)
}

func buildAppNameForm(target *string) *huh.Form {
	return newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter app name").
				Validate(validateAppName).
				Value(target),
		),
	)
}

func validateAppName(name string) error {
	return config.Project{AppName: name}.ValidateAppName()
}

func buildDevServerForm(target *bool) *huh.Form {
	return newForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Open and run in VS Code?").
				Affirmative("Yes").
				Negative("No").
				Value(target),
		),
	)
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
			return TickMsg{}
		}),
		m.Spinner.Tick,
		discoverPythonCmd(),
	)
}

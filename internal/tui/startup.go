package tui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/releases"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

func NewModel(engine scaffold.Options) *Model {
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
		project: config.Project{
			ProjectName:     "django-stack",
			CreateTemplates: false,
			InitializeGit:   false,
			DatabaseType:    "sqlite",
		},
		clock:           time.Now,
		engine:          engine,
		selectedStack:   config.StackStandard,
		spinner:         s,
		progress:        p,
		step:            stepSplashScreen,
		splashCountdown: 3,
		progressStatus:  "Initializing...",
		selectedOptions: []config.Feature{},
		completedSteps:  0,
	}

	m.catalog = releases.ForMachine(m.commandRunner, os.Getenv)
	m.projectNameForm = buildProjectNameForm(&m.project.ProjectName)
	m.pythonForm = buildPendingPythonForm(&m.project.SelectedPythonPath)
	m.djangoVersionForm = buildPendingDjangoForm(&m.project.DjangoVersion)
	m.databaseForm = buildDatabaseForm(&m.project.DatabaseType)
	m.stackForm = buildStackForm(&m.selectedStack)
	m.projectConfigForm = buildProjectConfigForm(&m.selectedOptions)
	m.appNameForm = buildAppNameForm(&m.project.AppName, &m.project.ProjectName, m.pythonNames)
	m.devServerForm = buildDevServerForm(&m.startDevServer)

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
	return config.Project{ProjectName: name}.ValidateProjectName(config.PythonNames{})
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

func projectConfigOptions() []huh.Option[config.Feature] {
	return []huh.Option[config.Feature]{
		huh.NewOption("Initialize Git Repository", config.FeatureGit),
		huh.NewOption("Django REST Framework API", config.FeatureRestFramework),
		huh.NewOption("Django Ninja API", config.FeatureNinja),
		huh.NewOption("Django AllAuth (authentication)", config.FeatureAllAuth),
		huh.NewOption("Ruff (Linter + Formatter)", config.FeatureRuff),
		huh.NewOption("Mypy (Type Checking)", config.FeatureMypy),
		huh.NewOption("Pre-commit Hooks (requires Git)", config.FeaturePreCommit),
		huh.NewOption("Multi-env Settings (dev/staging/prod)", config.FeatureMultiEnv),
		huh.NewOption("Typed Python (pydantic)", config.FeaturePydanticConfig),
		huh.NewOption("Structured Logging (structlog)", config.FeatureStructlog),
		huh.NewOption("Error Tracking (Sentry)", config.FeatureSentry),
	}
}

func validateOneAPIFramework(selected []config.Feature) error {
	return config.ValidateFeatures(selected)
}

func buildProjectConfigForm(target *[]config.Feature) *huh.Form {
	options := projectConfigOptions()

	return newForm(
		huh.NewGroup(
			huh.NewMultiSelect[config.Feature]().
				Title("Project Configuration" + "\n").
				Options(options...).
				Limit(len(options)).
				Validate(validateOneAPIFramework).
				Value(target),
		),
	)
}

func buildAppNameForm(target, projectName *string, names func() config.PythonNames) *huh.Form {
	return newForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter app name").
				Validate(func(name string) error { return validateAppName(name, *projectName, names()) }).
				Value(target),
		),
	)
}

func validateAppName(name, projectName string, names config.PythonNames) error {
	return config.Project{AppName: name, ProjectName: projectName}.ValidateAppName(names)
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
			return tickMsg{}
		}),
		m.spinner.Tick,
		m.discoverPythonCmd(),
	)
}

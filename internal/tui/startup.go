package tui

import (
	"os"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

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
		progress.WithColors(lipgloss.Color("#aa3bff"), lipgloss.Color("#f0a3ff")),
		progress.WithWidth(50),
		progress.WithFillCharacters('●', '○'),
		progress.WithoutPercentage(),
	)

	m := &Model{
		choices: config.Choices{
			Name:     "django-stack",
			Stack:    config.StackStandard,
			Database: "sqlite",
		},
		clock:           time.Now,
		engine:          engine,
		spinner:         s,
		progress:        p,
		step:            stepSplashScreen,
		splashCountdown: 3,
		progressStatus:  "Initializing...",
		selectedOptions: []config.AddOn{},
		completedSteps:  0,
	}

	m.catalog = releases.ForMachine(m.commandRunner, os.Getenv)
	m.projectNameForm = buildProjectNameForm(&m.choices.Name)
	m.pythonForm = buildPendingPythonForm(&m.resolved.PythonPath)
	m.djangoVersionForm = buildPendingDjangoForm(&m.choices.Django)
	m.databaseForm = buildDatabaseForm(&m.choices.Database)
	m.stackForm = buildStackForm(&m.choices.Stack)
	m.projectConfigForm = buildProjectConfigForm(&m.selectedOptions)
	m.appNameForm = buildAppNameForm(&m.choices.App, &m.choices.Name, m.pythonNames)
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
	return config.Choices{Name: name}.ValidateProjectName(config.PythonNames{})
}

func buildDatabaseForm(target *config.Database) *huh.Form {
	databases := config.Databases()
	options := make([]huh.Option[config.Database], 0, len(databases))
	for _, database := range databases {
		options = append(options, huh.NewOption(database.Spec().Option, database))
	}
	return selectForm("Choose Database", "", options, target, nil)
}

func projectConfigOptions() []huh.Option[config.AddOn] {
	addOns := config.AddOns()
	options := make([]huh.Option[config.AddOn], len(addOns))
	for i, addOn := range addOns {
		options[i] = huh.NewOption(addOn.Spec().Label, addOn)
	}
	return options
}

func buildProjectConfigForm(target *[]config.AddOn) *huh.Form {
	options := projectConfigOptions()
	title := "Project Configuration\n"

	return newForm(
		huh.NewGroup(
			huh.NewMultiSelect[config.AddOn]().
				Title(title).
				Options(options...).
				Height(len(options) + lipgloss.Height(title)).
				Limit(len(options)).
				Validate(config.ValidateAddOns).
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
	return config.Choices{App: name, Name: projectName}.ValidateAppName(names)
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
		tea.RequestBackgroundColor,
		m.discoverPythonCmd(),
	)
}

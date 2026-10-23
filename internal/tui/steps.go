package tui

import (
	"fmt"
	"sync"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

type stepSpec struct {
	title string

	position int

	question bool
	text     bool

	previous    wizardStep
	hasPrevious bool

	form func(m *Model) **huh.Form

	rebuild func(m *Model) *huh.Form

	next func(m *Model) []tea.Cmd

	footer footerText

	busyFooter footerText

	footerNext func(m *Model) string
}

const totalFlowSteps = 8

var stepsOnce struct {
	sync.Once
	table map[wizardStep]stepSpec
}

func steps() map[wizardStep]stepSpec {
	stepsOnce.Do(func() { stepsOnce.table = stepTable() })
	return stepsOnce.table
}

var (
	selectKeys = []string{"↑↓ move", "←→ step", "⏎ select", "ctrl+c quit"}
	exitKeys   = []string{"⏎ exit", "q exit"}
)

func stepTable() map[wizardStep]stepSpec {
	return map[wizardStep]stepSpec{
		stepSplashScreen: {title: "Get started", footer: footerText{
			keys: []string{"any key start", "ctrl+c quit"},
			hint: "Press any key to skip the countdown.",
			next: "Naming your project.",
		}},
		stepProjectName: {
			title: "Django Project Configuration", position: 1, question: true, text: true,
			footer: footerText{
				keys: []string{"type to edit", "⏎ next", "ctrl+c quit"},
				hint: "The arrow keys move the cursor inside the field.",
				next: "Choosing the Python interpreter to build the virtualenv with.",
			},
			form: func(m *Model) **huh.Form { return &m.projectNameForm },
			rebuild: func(m *Model) *huh.Form {
				m.projectNameForm = buildProjectNameForm(&m.project.ProjectName)
				return m.projectNameForm
			},
			next: func(m *Model) []tea.Cmd {
				m.step = stepPythonSelection
				return []tea.Cmd{m.pythonForm.Init()}
			},
		},
		stepPythonSelection: {
			title: "Python Selection", position: 2, question: true, previous: stepProjectName, hasPrevious: true,
			footer: footerText{
				keys: selectKeys,
				hint: "← returns to the project name; answers are kept.",
				next: "Choosing a Django release compatible with it.",
			},
			form: func(m *Model) **huh.Form { return &m.pythonForm },
			rebuild: func(m *Model) *huh.Form {
				if len(m.availablePythonOptions) == 0 {
					m.pythonForm = buildUnavailablePythonForm(&m.project.SelectedPythonPath, m.pythonDiscoveryNote)
				} else {
					m.pythonForm = m.buildPythonForm()
				}
				return m.pythonForm
			},
			next: func(m *Model) []tea.Cmd {
				m.syncSelectedPythonOption()
				m.step = stepDjangoVersion
				m.djangoVersionForm = buildPendingDjangoForm(&m.project.DjangoVersion)
				m.syncFormWidths()
				return []tea.Cmd{m.fetchDjangoOptionsCmd(), m.djangoVersionForm.Init()}
			},
		},
		stepDjangoVersion: {
			title: "Django Version Selection", position: 3, question: true, previous: stepPythonSelection, hasPrevious: true,
			footer: footerText{
				keys: selectKeys,
				hint: "← returns to the interpreter and refetches this list.",
				next: "Choosing the database.",
			},
			form: func(m *Model) **huh.Form { return &m.djangoVersionForm },
			rebuild: func(m *Model) *huh.Form {
				if len(m.availableDjangoOptions) == 0 {
					m.djangoVersionForm = buildUnavailableDjangoForm(&m.project.DjangoVersion, m.djangoVersionNote)
				} else {
					m.djangoVersionForm = m.buildDjangoVersionForm()
				}
				return m.djangoVersionForm
			},
			next: func(m *Model) []tea.Cmd {
				m.step = stepDatabase
				return []tea.Cmd{m.databaseForm.Init()}
			},
		},
		stepDatabase: {
			title: "Database Selection", position: 4, question: true, previous: stepDjangoVersion, hasPrevious: true,
			footer: footerText{
				keys: selectKeys,
				hint: "Docker options generate a compose file and wait for the container.",
				next: "Choosing the stack the project is built from.",
			},
			form: func(m *Model) **huh.Form { return &m.databaseForm },
			rebuild: func(m *Model) *huh.Form {
				m.databaseForm = buildDatabaseForm(&m.project.DatabaseType)
				return m.databaseForm
			},
			next: func(m *Model) []tea.Cmd {
				if m.project.DatabaseType.UsesDocker() {
					return m.beginDockerCheck()
				}
				m.step = stepStack
				return []tea.Cmd{m.stackForm.Init()}
			},
		},
		stepDockerCheck: {
			title: "Docker Check", position: 4, question: true, previous: stepDatabase, hasPrevious: true,
			footer: footerText{
				keys: selectKeys,
				hint: "Nothing has been written to disk yet, so retrying costs nothing.",
				next: "Choosing the stack the project is built from.",
			},
			busyFooter: footerText{
				keys: []string{"ctrl+c quit"},
				hint: "Asking Docker whether it is running.",
				next: "Starting the database container once Docker answers.",
			},
			form: func(m *Model) **huh.Form { return &m.dockerForm },
			next: (*Model).applyDockerRemedy,
		},
		stepStack: {
			title: "Stack Selection", position: 5, question: true, previous: stepDatabase, hasPrevious: true,
			footer: footerText{
				keys: selectKeys,
				hint: "This decides which HTML templates the project is built from.",
				next: "Choosing the add-ons to wire in.",
			},
			form: func(m *Model) **huh.Form { return &m.stackForm },
			rebuild: func(m *Model) *huh.Form {
				m.stackForm = buildStackForm(&m.project.Stack)
				return m.stackForm
			},
			next: func(m *Model) []tea.Cmd {
				m.step = stepProjectConfig
				m.projectConfigForm = buildProjectConfigForm(&m.selectedOptions)
				m.syncFormWidths()
				return []tea.Cmd{m.projectConfigForm.Init()}
			},
		},
		stepProjectConfig: {
			title: "Project Features", position: 6, question: true, previous: stepStack, hasPrevious: true,
			footer: footerText{
				keys: []string{"↑↓ move", "x toggle", "←→ step", "⏎ confirm", "ctrl+c quit"},
				hint: "Space or x toggles an add-on.",
				next: "Naming the app, if the add-ons you picked need one.",
			},
			form: func(m *Model) **huh.Form { return &m.projectConfigForm },
			rebuild: func(m *Model) *huh.Form {
				m.projectConfigForm = buildProjectConfigForm(&m.selectedOptions)
				return m.projectConfigForm
			},
			next: func(m *Model) []tea.Cmd {
				m.processFormData()
				if m.project.NeedsAppName() {
					m.step = stepAppName
					return []tea.Cmd{m.appNameForm.Init()}
				}
				return m.beginGeneration()
			},
		},
		stepAppName: {
			title: "App Configuration", position: 7, question: true, text: true, previous: stepProjectConfig, hasPrevious: true,
			footer: footerText{
				keys: []string{"⏎ next", "⇧⇥ back", "ctrl+c quit"},
				hint: "The arrow keys move the cursor; shift+tab reopens the feature list.",
				next: "Building the project. Nothing else is asked after this.",
			},
			form: func(m *Model) **huh.Form { return &m.appNameForm },
			rebuild: func(m *Model) *huh.Form {
				m.appNameForm = buildAppNameForm(&m.project.AppName, &m.project.ProjectName, m.pythonNames)
				return m.appNameForm
			},
			next: (*Model).beginGeneration,
		},
		stepSetup: {
			title: "Project Initialization", position: 8,
			footer: footerText{
				keys: []string{"ctrl+c cancel"},
				hint: "Quitting cancels the run and stops every child process.",
			},
			busyFooter: footerText{
				keys: []string{"ctrl+c force quit"},
				hint: "Waiting for the run to stop and clean up after itself.",
				next: "The partial project and its database are being removed.",
			},
			footerNext: func(m *Model) string { return fmt.Sprintf("%d of %d steps done.", m.completedSteps, m.totalSteps) },
		},
		stepDevServerPrompt: {
			title: "Project Setup Complete!",
			footer: footerText{
				keys: []string{"←→ move", "⏎ confirm", "ctrl+c quit"},
				hint: "Yes opens a fresh VS Code workspace and starts the dev task.",
				next: "Everything is scaffolded, linted and formatted.",
			},
			form: func(m *Model) **huh.Form { return &m.devServerForm },
			next: func(m *Model) []tea.Cmd {
				if !m.startDevServer {
					m.step = stepComplete
					return nil
				}
				m.done = true
				return []tea.Cmd{tea.Sequence(m.startDevelopmentEnvironment(), tea.Quit)}
			},
		},
		stepComplete: {title: "All Done!", footer: footerText{
			keys: exitKeys,
			hint: "You can close this now.",
			next: "Start the server when you are ready.",
		}},
	}
}

func screenTitle(step wizardStep) string { return steps()[step].title }

func stepPosition(step wizardStep) (int, bool) {
	position := steps()[step].position
	return position, position > 0
}

func previousStep(step wizardStep) (wizardStep, bool) {
	spec := steps()[step]
	return spec.previous, spec.hasPrevious
}

func isQuestionStep(step wizardStep) bool { return steps()[step].question }

func isTextStep(step wizardStep) bool { return steps()[step].text }

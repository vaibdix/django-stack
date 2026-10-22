package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
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
}

const totalFlowSteps = 8

var steps map[wizardStep]stepSpec

func init() {
	steps = stepTable()
	screenTitles = project(func(s stepSpec) (string, bool) { return s.title, true })
	stepPositions = project(func(s stepSpec) (int, bool) { return s.position, s.position > 0 })
	previousStep = project(func(s stepSpec) (wizardStep, bool) { return s.previous, s.hasPrevious })
}

func stepTable() map[wizardStep]stepSpec {
	return map[wizardStep]stepSpec{
		stepSplashScreen: {title: "Get started"},
		stepProjectName: {
			title: "Django Project Configuration", position: 1, question: true, text: true,
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
			form: func(m *Model) **huh.Form { return &m.dockerForm },
			next: (*Model).applyDockerRemedy,
		},
		stepStack: {
			title: "Stack Selection", position: 5, question: true, previous: stepDatabase, hasPrevious: true,
			form: func(m *Model) **huh.Form { return &m.stackForm },
			rebuild: func(m *Model) *huh.Form {
				m.stackForm = buildStackForm(&m.selectedStack)
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
			form: func(m *Model) **huh.Form { return &m.appNameForm },
			rebuild: func(m *Model) *huh.Form {
				m.appNameForm = buildAppNameForm(&m.project.AppName, &m.project.ProjectName, m.pythonNames)
				return m.appNameForm
			},
			next: (*Model).beginGeneration,
		},
		stepSetup: {title: "Project Initialization", position: 8},
		stepDevServerPrompt: {
			title: "Project Setup Complete!",
			form:  func(m *Model) **huh.Form { return &m.devServerForm },
			next: func(m *Model) []tea.Cmd {
				if !m.startDevServer {
					m.step = stepComplete
					return nil
				}
				m.startDevelopmentEnvironment()
				m.done = true
				return []tea.Cmd{func() tea.Msg { return tea.KeyMsg{Type: tea.KeyEnter} }}
			},
		},
		stepComplete: {title: "All Done!"},
	}
}

var (
	screenTitles  map[wizardStep]string
	stepPositions map[wizardStep]int
	previousStep  map[wizardStep]wizardStep
)

func isQuestionStep(step wizardStep) bool { return steps[step].question }

func isTextStep(step wizardStep) bool { return steps[step].text }

func project[V any](pick func(stepSpec) (V, bool)) map[wizardStep]V {
	out := make(map[wizardStep]V, len(steps))
	for step, spec := range steps {
		if value, ok := pick(spec); ok {
			out[step] = value
		}
	}
	return out
}

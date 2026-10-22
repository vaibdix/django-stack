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

	previous    Step
	hasPrevious bool

	form func(m *Model) **huh.Form

	rebuild func(m *Model) *huh.Form

	next func(m *Model) []tea.Cmd
}

const totalFlowSteps = 8

var steps map[Step]stepSpec

func init() {
	steps = stepTable()
	screenTitles = project(func(s stepSpec) (string, bool) { return s.title, true })
	stepPositions = project(func(s stepSpec) (int, bool) { return s.position, s.position > 0 })
	previousStep = project(func(s stepSpec) (Step, bool) { return s.previous, s.hasPrevious })
}

func stepTable() map[Step]stepSpec {
	return map[Step]stepSpec{
		StepSplashScreen: {title: "Get started"},
		StepProjectName: {
			title: "Django Project Configuration", position: 1, question: true, text: true,
			form: func(m *Model) **huh.Form { return &m.ProjectNameForm },
			rebuild: func(m *Model) *huh.Form {
				m.ProjectNameForm = buildProjectNameForm(&m.ProjectName)
				return m.ProjectNameForm
			},
			next: func(m *Model) []tea.Cmd {
				m.Step = StepPythonSelection
				return []tea.Cmd{m.PythonForm.Init()}
			},
		},
		StepPythonSelection: {
			title: "Python Selection", position: 2, question: true, previous: StepProjectName, hasPrevious: true,
			form: func(m *Model) **huh.Form { return &m.PythonForm },
			rebuild: func(m *Model) *huh.Form {
				if len(m.AvailablePythonOptions) == 0 {
					m.PythonForm = buildUnavailablePythonForm(&m.SelectedPythonPath, m.PythonDiscoveryNote)
				} else {
					m.PythonForm = m.buildPythonForm()
				}
				return m.PythonForm
			},
			next: func(m *Model) []tea.Cmd {
				m.syncSelectedPythonOption()
				m.Step = StepDjangoVersion
				m.DjangoVersionForm = buildPendingDjangoForm(&m.DjangoVersion)
				m.syncFormWidths()
				return []tea.Cmd{m.fetchDjangoOptionsCmd(), m.DjangoVersionForm.Init()}
			},
		},
		StepDjangoVersion: {
			title: "Django Version Selection", position: 3, question: true, previous: StepPythonSelection, hasPrevious: true,
			form: func(m *Model) **huh.Form { return &m.DjangoVersionForm },
			rebuild: func(m *Model) *huh.Form {
				if len(m.AvailableDjangoOptions) == 0 {
					m.DjangoVersionForm = buildUnavailableDjangoForm(&m.DjangoVersion, m.DjangoVersionNote)
				} else {
					m.DjangoVersionForm = m.buildDjangoVersionForm()
				}
				return m.DjangoVersionForm
			},
			next: func(m *Model) []tea.Cmd {
				m.Step = StepDatabase
				return []tea.Cmd{m.DatabaseForm.Init()}
			},
		},
		StepDatabase: {
			title: "Database Selection", position: 4, question: true, previous: StepDjangoVersion, hasPrevious: true,
			form: func(m *Model) **huh.Form { return &m.DatabaseForm },
			rebuild: func(m *Model) *huh.Form {
				m.DatabaseForm = buildDatabaseForm(&m.DatabaseType)
				return m.DatabaseForm
			},
			next: func(m *Model) []tea.Cmd {
				if m.DatabaseType.UsesDocker() {
					return m.beginDockerCheck()
				}
				m.Step = StepStack
				return []tea.Cmd{m.StackForm.Init()}
			},
		},
		StepDockerCheck: {
			title: "Docker Check", position: 4, question: true, previous: StepDatabase, hasPrevious: true,
			form: func(m *Model) **huh.Form { return &m.DockerForm },
			next: (*Model).applyDockerRemedy,
		},
		StepStack: {
			title: "Stack Selection", position: 5, question: true, previous: StepDatabase, hasPrevious: true,
			form: func(m *Model) **huh.Form { return &m.StackForm },
			rebuild: func(m *Model) *huh.Form {
				m.StackForm = buildStackForm(&m.SelectedStack)
				return m.StackForm
			},
			next: func(m *Model) []tea.Cmd {
				m.Step = StepProjectConfig
				m.ProjectConfigForm = buildProjectConfigForm(&m.SelectedOptions)
				m.syncFormWidths()
				return []tea.Cmd{m.ProjectConfigForm.Init()}
			},
		},
		StepProjectConfig: {
			title: "Project Features", position: 6, question: true, previous: StepStack, hasPrevious: true,
			form: func(m *Model) **huh.Form { return &m.ProjectConfigForm },
			rebuild: func(m *Model) *huh.Form {
				m.ProjectConfigForm = buildProjectConfigForm(&m.SelectedOptions)
				return m.ProjectConfigForm
			},
			next: func(m *Model) []tea.Cmd {
				m.ProcessFormData()
				if m.NeedsAppName() {
					m.Step = StepAppName
					return []tea.Cmd{m.AppNameForm.Init()}
				}
				return m.beginGeneration()
			},
		},
		StepAppName: {
			title: "App Configuration", position: 7, question: true, text: true, previous: StepProjectConfig, hasPrevious: true,
			form: func(m *Model) **huh.Form { return &m.AppNameForm },
			rebuild: func(m *Model) *huh.Form {
				m.AppNameForm = buildAppNameForm(&m.AppName, &m.ProjectName, m.pythonNames)
				return m.AppNameForm
			},
			next: (*Model).beginGeneration,
		},
		StepSetup: {title: "Project Initialization", position: 8},
		StepDevServerPrompt: {
			title: "Project Setup Complete!",
			form:  func(m *Model) **huh.Form { return &m.DevServerForm },
			next: func(m *Model) []tea.Cmd {
				if !m.StartDevServer {
					m.Step = StepComplete
					return nil
				}
				m.startDevelopmentEnvironment()
				m.Done = true
				return []tea.Cmd{func() tea.Msg { return tea.KeyMsg{Type: tea.KeyEnter} }}
			},
		},
		StepComplete: {title: "All Done!"},
	}
}

var (
	screenTitles  map[Step]string
	stepPositions map[Step]int
	previousStep  map[Step]Step
)

func isQuestionStep(step Step) bool { return steps[step].question }

func isTextStep(step Step) bool { return steps[step].text }

func project[V any](pick func(stepSpec) (V, bool)) map[Step]V {
	out := make(map[Step]V, len(steps))
	for step, spec := range steps {
		if value, ok := pick(spec); ok {
			out[step] = value
		}
	}
	return out
}

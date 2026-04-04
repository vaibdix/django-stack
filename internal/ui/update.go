package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var spinnerCmd tea.Cmd
	m.Spinner, spinnerCmd = m.Spinner.Update(msg)
	cmds = append(cmds, spinnerCmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyCtrlC || keyMsg.String() == "q" {
			if !m.Done && m.Error == nil {
				return m, tea.Quit
			}
		}
	}
	if m.Error != nil || m.Done {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.Type == tea.KeyEnter || keyMsg.Type == tea.KeyCtrlC || keyMsg.String() == "q" || keyMsg.String() == "esc" {
				return m, tea.Quit
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Progress.Width = m.Width - 4 - 4
		if m.Progress.Width > 80 {
			m.Progress.Width = 80
		}
		if m.Progress.Width < 20 {
			m.Progress.Width = 20
		}
		return m, nil

	case TickMsg:
		if m.Step == StepSplashScreen {
			m.SplashCountdown--
			if m.SplashCountdown <= 0 {
				m.Step = StepProjectName
				cmds = append(cmds, m.ProjectNameForm.Init())
			} else {
				cmds = append(cmds, tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
					return TickMsg{}
				}))
			}
		}
		return m, tea.Batch(cmds...)

	case BlinkMsg:
		if time.Since(m.BlinkTimer) > time.Second/2 {
			m.EyesOpen = !m.EyesOpen
			m.BlinkTimer = time.Now()
		}
		cmds = append(cmds, BlinkCmd())
		return m, tea.Batch(cmds...)

	case ProjectProgressMsg:
		if m.Step == StepSetup {
			cmd := m.Progress.SetPercent(msg.Percent)
			m.ProgressStatus = msg.Status
			m.AppendStepMessage("PROGRESS: " + msg.Status)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case ProjectCreationDoneMsg:
		if m.Step == StepSetup {
			if msg.Err != nil {
				m.Error = msg.Err
				m.ProgressStatus = "Error during project setup!"
				return m, nil
			}
			cmd := m.Progress.SetPercent(1.0)
			m.ProgressStatus = "Django project setup complete!"
			m.AppendStepMessage("Django project setup complete!")
			m.Step = StepDevServerPrompt
			cmds = append(cmds, cmd, m.DevServerForm.Init())
		}
		return m, tea.Batch(cmds...)

	case progress.FrameMsg:
		progressModel, cmd := m.Progress.Update(msg)
		m.Progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	activeForm := m.GetActiveForm()
	if activeForm != nil && activeForm.State != huh.StateCompleted {
		formModel, formCmd := activeForm.Update(msg)
		if castedForm, ok := formModel.(*huh.Form); ok {
			switch m.Step {
			case StepProjectName:
				m.ProjectNameForm = castedForm
			case StepPythonSelection:
				m.PythonForm = castedForm
			case StepDjangoVersion:
				m.DjangoVersionForm = castedForm
			case StepDatabase:
				m.DatabaseForm = castedForm
			case StepProjectConfig:
				m.ProjectConfigForm = castedForm
			case StepAppName:
				m.AppNameForm = castedForm
			case StepDevServerPrompt:
				m.DevServerForm = castedForm
			}
		}
		cmds = append(cmds, formCmd)
	} else if activeForm != nil && activeForm.State == huh.StateCompleted {
		switch m.Step {
		case StepProjectName:
			m.Step = StepPythonSelection
			m.RefreshPythonOptions()
			cmds = append(cmds, m.PythonForm.Init())
		case StepPythonSelection:
			m.syncSelectedPythonOption()
			m.RefreshDjangoOptions()
			m.Step = StepDjangoVersion
			cmds = append(cmds, m.DjangoVersionForm.Init())
		case StepDjangoVersion:
			m.Step = StepDatabase
			cmds = append(cmds, m.DatabaseForm.Init())
		case StepDatabase:
			m.Step = StepProjectConfig
			cmds = append(cmds, m.ProjectConfigForm.Init())
		case StepProjectConfig:
			m.ProcessFormData()
			if m.CreateAppTemplates || m.SetupRestFramework || m.SetupHTMX || m.SetupAllAuth {
				m.Step = StepAppName
				cmds = append(cmds, m.AppNameForm.Init())
			} else {
				m.Step = StepSetup
				m.TotalSteps = m.CalculateTotalSteps()
				m.ProgressStatus = "Starting project setup..."
				m.Progress.SetPercent(0.0)
				go func() {
					_ = CreateProject(m)
				}()
				cmds = append(cmds,
					m.Spinner.Tick,
					func() tea.Msg {
						return ProjectProgressMsg{
							Percent: 0.0,
							Status:  "Initializing project setup...",
						}
					},
					m.Progress.SetPercent(0.0),
					func() tea.Msg {
						return progress.FrameMsg{}
					},
				)
			}
		case StepAppName:
			m.Step = StepSetup
			m.TotalSteps = m.CalculateTotalSteps()
			m.ProgressStatus = "Starting project setup..."
			m.Progress.SetPercent(0.0)
			go func() {
				_ = CreateProject(m)
			}()
			cmds = append(cmds,
				m.Spinner.Tick,
				func() tea.Msg {
					return ProjectProgressMsg{
						Percent: 0.0,
						Status:  "Initializing project setup...",
					}
				},
				m.Progress.SetPercent(0.0),
				func() tea.Msg {
					return progress.FrameMsg{}
				},
			)
		case StepDevServerPrompt:
			if m.StartDevServer {
				m.StartDevelopmentEnvironment()
				m.Done = true
				cmds = append(cmds, func() tea.Msg {
					return tea.KeyMsg{Type: tea.KeyEnter}
				})
			} else {
				m.Step = StepComplete
			}
		}
		return m, tea.Batch(cmds...)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) ProcessFormData() {
	m.CreateTemplates = false
	m.CreateAppTemplates = false
	m.InitializeGit = false
	m.SetupTailwind = false
	m.SetupRestFramework = false
	m.SetupHTMX = false
	m.SetupAllAuth = false
	m.SetupRuff = false
	m.SetupMypy = false
	m.SetupPreCommit = false
	m.SetupMultiEnv = false

	for _, opt := range m.SelectedOptions {
		switch opt {
		case "Standard Django Project":
			m.CreateTemplates = true
			m.CreateAppTemplates = true
		case "Initialize Git":
			m.InitializeGit = true
		case "Tailwind":
			m.SetupTailwind = true
		case "REST Framework":
			m.SetupRestFramework = true
		case "HTMX":
			m.SetupHTMX = true
		case "AllAuth":
			m.SetupAllAuth = true
		case "Ruff":
			m.SetupRuff = true
		case "Mypy":
			m.SetupMypy = true
		case "Pre-commit":
			m.SetupPreCommit = true
		case "Multi-env":
			m.SetupMultiEnv = true
		}
	}

	// Pre-commit requires Git — auto-enable if needed
	if m.SetupPreCommit && !m.InitializeGit {
		m.InitializeGit = true
	}
	m.AppendStepMessage("Project name: " + m.ProjectName)
	if m.SelectedPythonPath != "" {
		m.AppendStepMessage("Python interpreter: " + m.SelectedPythonPath + " (" + m.SelectedPythonVersion + ")")
	}
	m.AppendStepMessage("Django version: " + m.DjangoVersion)
	m.AppendStepMessage("Database: " + m.DatabaseType)
	if m.AppName != "" {
		m.AppendStepMessage("App name: " + m.AppName)
	}
	m.AppendStepMessage(fmt.Sprintf("Selected options: %v", m.SelectedOptions))
}

func BlinkCmd() tea.Cmd {
	return tea.Tick(time.Second/2, func(_ time.Time) tea.Msg {
		return BlinkMsg{}
	})
}

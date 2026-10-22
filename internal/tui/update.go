package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/scaffold"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var spinnerCmd tea.Cmd
	m.Spinner, spinnerCmd = m.Spinner.Update(msg)
	cmds = append(cmds, spinnerCmd)

	if cmd, handled := m.handleExitKeys(msg); handled {
		return m, cmd
	}

	if _, ok := msg.(tea.KeyMsg); ok && m.Step == StepSplashScreen {
		m.SplashCountdown = 0
		m.Step = StepProjectName
		return m, tea.Batch(append(cmds, m.ProjectNameForm.Init())...)
	}

	if stepCmds, handled := m.handleStepKeys(msg); handled {
		return m, tea.Batch(append(cmds, stepCmds...)...)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Progress.Width = innerWidth(msg.Width)
		m.syncFormWidths()
		return m, nil

	case TickMsg:
		if m.Step == StepSplashScreen && m.SplashCountdown > 0 {
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

	case pythonOptionsMsg:
		m.applyPythonOptions(msg.options, msg.err)
		m.syncFormWidths()
		if m.Step == StepPythonSelection {
			cmds = append(cmds, m.PythonForm.Init())
		}
		return m, tea.Batch(cmds...)

	case dockerStatusMsg:
		cmds = append(cmds, m.applyDockerStatus(msg)...)
		return m, tea.Batch(cmds...)

	case djangoOptionsMsg:
		if !m.isCurrentDjangoReply(msg) {
			return m, nil
		}
		m.applyDjangoOptions(msg.Options, msg.Note)
		m.syncFormWidths()
		if m.Step == StepDjangoVersion {
			cmds = append(cmds, m.DjangoVersionForm.Init())
		}
		return m, tea.Batch(cmds...)

	case BlinkMsg:
		if time.Since(m.BlinkTimer) > time.Second/2 {
			m.EyesOpen = !m.EyesOpen
			m.BlinkTimer = time.Now()
		}
		cmds = append(cmds, BlinkCmd())
		return m, tea.Batch(cmds...)

	case scaffoldStepMsg:
		m.StepMessages = append(m.StepMessages, string(msg))
		cmds = append(cmds, m.waitForScaffoldEvent())
		return m, tea.Batch(cmds...)

	case scaffoldProgressMsg:
		m.ProgressStatus = string(msg)
		m.StepMessages = append(m.StepMessages, string(msg))
		cmds = append(cmds, m.Progress.SetPercent(m.advanceProgress()), m.waitForScaffoldEvent())
		return m, tea.Batch(cmds...)

	case scaffoldDoneMsg:
		if m.Step == StepSetup {
			if msg.Err != nil {
				m.Error = msg.Err
				m.ProgressStatus = "Error during project setup!"
				return m, nil
			}
			cmd := m.Progress.SetPercent(1.0)
			m.ProgressStatus = "Django project setup complete!"
			m.StepMessages = append(m.StepMessages, "Django project setup complete!")
			m.Step = StepDevServerPrompt
			cmds = append(cmds, cmd, m.DevServerForm.Init())
		}
		return m, tea.Batch(cmds...)

	case progress.FrameMsg:
		progressModel, cmd := m.Progress.Update(msg)
		if updated, ok := progressModel.(progress.Model); ok {
			m.Progress = updated
		}
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	return m, tea.Batch(append(cmds, m.updateActiveForm(msg)...)...)
}

func (m *Model) updateActiveForm(msg tea.Msg) []tea.Cmd {
	activeForm := m.GetActiveForm()
	if activeForm == nil {
		return nil
	}

	if activeForm.State == huh.StateCompleted {
		return m.advanceFromCompletedForm()
	}

	formModel, formCmd := activeForm.Update(msg)
	if castedForm, ok := formModel.(*huh.Form); ok {
		m.setActiveForm(castedForm)
		activeForm = castedForm
	}

	cmds := []tea.Cmd{formCmd}
	if activeForm.State == huh.StateCompleted {
		cmds = append(cmds, m.advanceFromCompletedForm()...)
	}
	return cmds
}

func (m *Model) setActiveForm(form *huh.Form) {
	if slot := steps[m.Step].form; slot != nil {
		*slot(m) = form
	}
}

func (m *Model) advanceFromCompletedForm() []tea.Cmd {
	if next := steps[m.Step].next; next != nil {
		return next(m)
	}
	return nil
}

func (m *Model) beginGeneration() []tea.Cmd {
	m.Step = StepSetup
	if m.SetupTailwind {

		m.NodeManager, _ = scaffold.DetectNodeManager(context.Background(), m.runner())
	}
	m.TotalSteps = m.CalculateTotalSteps()
	m.CompletedSteps = 0
	m.ProgressStatus = "Starting project setup..."

	return []tea.Cmd{
		m.Spinner.Tick,
		m.Progress.SetPercent(0.0),
		m.StartGeneration(),
	}
}

func (m *Model) handleExitKeys(msg tea.Msg) (tea.Cmd, bool) {
	keyMsg, ok := msg.(tea.KeyMsg)

	if ok && keyMsg.Type == tea.KeyCtrlC {
		if m.generationRunning() && !m.Cancelling {
			return m.cancelAndWait(), true
		}
		m.CancelGeneration()
		return tea.Quit, true
	}

	if m.Error != nil || m.Done {
		if ok && isDismissKey(keyMsg) {
			return tea.Quit, true
		}
		return nil, true
	}

	if ok && m.atTerminalScreen() && isDismissKey(keyMsg) {
		return tea.Quit, true
	}

	return nil, false
}

func (m *Model) atTerminalScreen() bool {
	return m.Done || m.Error != nil || m.Step == StepComplete
}

func isDismissKey(k tea.KeyMsg) bool {
	switch k.Type {
	case tea.KeyEnter, tea.KeyCtrlC:
		return true
	default:
		return k.String() == "q" || k.String() == "esc"
	}
}

func (m *Model) ProcessFormData() {
	m.ApplyFeatures(m.selectedFeatures())

	m.StepMessages = nil
	m.StepMessages = append(m.StepMessages, "Project name: "+m.ProjectName)
	if m.SelectedPythonPath != "" {
		m.StepMessages = append(m.StepMessages, "Python interpreter: "+m.SelectedPythonPath+" ("+m.SelectedPythonVersion+")")
	}
	m.StepMessages = append(m.StepMessages, "Django version: "+m.DjangoVersion)
	m.StepMessages = append(m.StepMessages, "Database: "+string(m.DatabaseType))
	m.StepMessages = append(m.StepMessages, "Stack: "+m.SelectedStack.Label())
	if m.AppName != "" {
		m.StepMessages = append(m.StepMessages, "App name: "+m.AppName)
	}
	m.StepMessages = append(m.StepMessages, fmt.Sprintf("Add-ons: %v", m.SelectedOptions))
}

func BlinkCmd() tea.Cmd {
	return tea.Tick(time.Second/2, func(_ time.Time) tea.Msg {
		return BlinkMsg{}
	})
}

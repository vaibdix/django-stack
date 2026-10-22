package tui

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
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	cmds = append(cmds, spinnerCmd)

	if cmd, handled := m.handleExitKeys(msg); handled {
		return m, cmd
	}

	if _, ok := msg.(tea.KeyMsg); ok && m.step == stepSplashScreen {
		m.splashCountdown = 0
		m.step = stepProjectName
		return m, tea.Batch(append(cmds, m.projectNameForm.Init())...)
	}

	if stepCmds, handled := m.handleStepKeys(msg); handled {
		return m, tea.Batch(append(cmds, stepCmds...)...)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = innerWidth(msg.Width)
		m.syncFormWidths()
		return m, nil

	case tickMsg:
		if m.step == stepSplashScreen && m.splashCountdown > 0 {
			m.splashCountdown--
			if m.splashCountdown <= 0 {
				m.step = stepProjectName
				cmds = append(cmds, m.projectNameForm.Init())
			} else {
				cmds = append(cmds, tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
					return tickMsg{}
				}))
			}
		}
		return m, tea.Batch(cmds...)

	case pythonOptionsMsg:
		m.applyPythonOptions(msg.options, msg.err)
		m.syncFormWidths()
		if m.step == stepPythonSelection {
			cmds = append(cmds, m.pythonForm.Init())
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
		if m.step == stepDjangoVersion {
			cmds = append(cmds, m.djangoVersionForm.Init())
		}
		return m, tea.Batch(cmds...)

	case toolsDecidedMsg:
		m.nodeManager = msg.NodeManager
		return m, m.waitForScaffoldEvent()

	case scaffoldStepMsg:
		m.stepMessages = append(m.stepMessages, string(msg))
		cmds = append(cmds, m.waitForScaffoldEvent())
		return m, tea.Batch(cmds...)

	case scaffoldProgressMsg:
		m.progressStatus = string(msg)
		m.stepMessages = append(m.stepMessages, string(msg))
		cmds = append(cmds, m.progress.SetPercent(m.advanceProgress()), m.waitForScaffoldEvent())
		return m, tea.Batch(cmds...)

	case scaffoldDoneMsg:
		if m.step == stepSetup {
			if msg.Err != nil {
				m.failure = msg.Err
				m.progressStatus = "Error during project setup!"
				return m, nil
			}
			cmd := m.progress.SetPercent(1.0)
			m.progressStatus = "Django project setup complete!"
			m.stepMessages = append(m.stepMessages, "Django project setup complete!")
			m.step = stepDevServerPrompt
			cmds = append(cmds, cmd, m.devServerForm.Init())
		}
		return m, tea.Batch(cmds...)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		if updated, ok := progressModel.(progress.Model); ok {
			m.progress = updated
		}
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	return m, tea.Batch(append(cmds, m.updateActiveForm(msg)...)...)
}

func (m *Model) updateActiveForm(msg tea.Msg) []tea.Cmd {
	activeForm := m.currentForm()
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
	if slot := steps[m.step].form; slot != nil {
		*slot(m) = form
	}
}

func (m *Model) advanceFromCompletedForm() []tea.Cmd {
	if next := steps[m.step].next; next != nil {
		return next(m)
	}
	return nil
}

func (m *Model) beginGeneration() []tea.Cmd {
	m.step = stepSetup
	m.totalSteps = m.calculateTotalSteps()
	m.completedSteps = 0
	m.progressStatus = "Starting project setup..."

	return []tea.Cmd{
		m.spinner.Tick,
		m.progress.SetPercent(0.0),
		m.startGeneration(),
	}
}

func (m *Model) handleExitKeys(msg tea.Msg) (tea.Cmd, bool) {
	keyMsg, ok := msg.(tea.KeyMsg)

	if ok && keyMsg.Type == tea.KeyCtrlC {
		if m.generationRunning() && !m.cancelling {
			return m.cancelAndWait(), true
		}
		m.cancelGeneration()
		return tea.Quit, true
	}

	if m.failure != nil || m.done {
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
	return m.done || m.failure != nil || m.step == stepComplete
}

func isDismissKey(k tea.KeyMsg) bool {
	switch k.Type {
	case tea.KeyEnter, tea.KeyCtrlC:
		return true
	default:
		return k.String() == "q" || k.String() == "esc"
	}
}

func (m *Model) processFormData() {
	m.project.ApplyFeatures(m.selectedFeatures())

	m.stepMessages = nil
	m.stepMessages = append(m.stepMessages, "Project name: "+m.project.ProjectName)
	if m.project.SelectedPythonPath != "" {
		m.stepMessages = append(m.stepMessages, "Python interpreter: "+m.project.SelectedPythonPath+" ("+m.project.SelectedPythonVersion+")")
	}
	m.stepMessages = append(m.stepMessages, "Django version: "+m.project.DjangoVersion)
	m.stepMessages = append(m.stepMessages, "Database: "+string(m.project.DatabaseType))
	m.stepMessages = append(m.stepMessages, "Stack: "+m.selectedStack.Label())
	if m.project.AppName != "" {
		m.stepMessages = append(m.stepMessages, "App name: "+m.project.AppName)
	}
	m.stepMessages = append(m.stepMessages, fmt.Sprintf("Add-ons: %v", m.selectedOptions))
}

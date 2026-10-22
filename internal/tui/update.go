package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)

	if started, ok := msg.(devEnvStartedMsg); ok {
		m.stepMessages = append(m.stepMessages, started...)
		return m, nil
	}
	if cmd, handled := m.handleExitKeys(msg); handled {
		return m, cmd
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		cmd = m.onKey(msg)
	case tea.WindowSizeMsg:
		m.onWindowSize(msg)
	case tickMsg:
		cmd = m.onTick()
	case pythonOptionsMsg, djangoOptionsMsg:
		cmd = m.onOptions(msg)
	case dockerStatusMsg:
		cmd = tea.Batch(m.applyDockerStatus(msg)...)
	case toolsDecidedMsg, scaffoldStepMsg, scaffoldProgressMsg, scaffoldDoneMsg:
		cmd = m.onScaffoldEvent(msg)
	case progress.FrameMsg:
		cmd = m.onProgressFrame(msg)
	default:
		cmd = tea.Batch(m.updateActiveForm(msg)...)
	}
	return m, tea.Batch(spinnerCmd, cmd)
}

func (m *Model) onKey(msg tea.KeyMsg) tea.Cmd {
	if m.step == stepSplashScreen {
		m.splashCountdown = 0
		m.step = stepProjectName
		return m.projectNameForm.Init()
	}
	if cmds, handled := m.handleStepKeys(msg); handled {
		return tea.Batch(cmds...)
	}
	return tea.Batch(m.updateActiveForm(msg)...)
}

func (m *Model) onWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.progress.Width = innerWidth(msg.Width)
	m.syncFormWidths()
}

func (m *Model) onTick() tea.Cmd {
	if m.step != stepSplashScreen || m.splashCountdown <= 0 {
		return nil
	}
	m.splashCountdown--
	if m.splashCountdown <= 0 {
		m.step = stepProjectName
		return m.projectNameForm.Init()
	}
	return tea.Tick(1*time.Second, func(_ time.Time) tea.Msg { return tickMsg{} })
}

func (m *Model) onOptions(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case pythonOptionsMsg:
		m.applyPythonOptions(msg.options, msg.err)
		m.syncFormWidths()
		if m.step == stepPythonSelection {
			return m.pythonForm.Init()
		}
	case djangoOptionsMsg:
		if !m.isCurrentDjangoReply(msg) {
			return nil
		}
		m.applyDjangoOptions(msg.Options, msg.Note)
		m.syncFormWidths()
		if m.step == stepDjangoVersion {
			return m.djangoVersionForm.Init()
		}
	}
	return nil
}

func (m *Model) onScaffoldEvent(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case toolsDecidedMsg:
		m.nodeManager = msg.NodeManager
		return m.waitForScaffoldEvent()
	case scaffoldStepMsg:
		m.stepMessages = append(m.stepMessages, string(msg))
		return m.waitForScaffoldEvent()
	case scaffoldProgressMsg:
		m.progressStatus = string(msg)
		m.stepMessages = append(m.stepMessages, string(msg))
		return tea.Batch(m.progress.SetPercent(m.advanceProgress()), m.waitForScaffoldEvent())
	case scaffoldDoneMsg:
		return m.onScaffoldDone(msg)
	}
	return nil
}

func (m *Model) onScaffoldDone(msg scaffoldDoneMsg) tea.Cmd {
	if m.step != stepSetup {
		return nil
	}
	if msg.Err != nil {
		m.failure = msg.Err
		m.progressStatus = "Error during project setup!"
		return nil
	}
	m.progressStatus = "Django project setup complete!"
	m.stepMessages = append(m.stepMessages, "Django project setup complete!")
	m.step = stepDevServerPrompt
	return tea.Batch(m.progress.SetPercent(1.0), m.devServerForm.Init())
}

func (m *Model) onProgressFrame(msg progress.FrameMsg) tea.Cmd {
	progressModel, cmd := m.progress.Update(msg)
	if updated, ok := progressModel.(progress.Model); ok {
		m.progress = updated
	}
	return cmd
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
	if slot := steps()[m.step].form; slot != nil {
		*slot(m) = form
	}
}

func (m *Model) advanceFromCompletedForm() []tea.Cmd {
	if next := steps()[m.step].next; next != nil {
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

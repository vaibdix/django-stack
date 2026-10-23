package tui

import (
	"time"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/vaibdix/django-stack/internal/config"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.update(msg)
	m.fitActiveForm()
	return m, cmd
}

func (m *Model) update(msg tea.Msg) tea.Cmd {
	var spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)

	if started, ok := msg.(devEnvStartedMsg); ok {
		m.stepMessages = append(m.stepMessages, started...)
		return nil
	}
	if cmd, handled := m.handleExitKeys(msg); handled {
		return cmd
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		cmd = m.onKey(msg)
	case tea.WindowSizeMsg, tea.BackgroundColorMsg:
		m.onTerminal(msg)
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
	return tea.Batch(spinnerCmd, cmd)
}

func (m *Model) onKey(msg tea.KeyPressMsg) tea.Cmd {
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

func (m *Model) onTerminal(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		lightBackground.Store(!msg.IsDark())
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.SetWidth(innerWidth(msg.Width))
		m.syncFormWidths()
	}
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
	var cmd tea.Cmd
	m.progress, cmd = m.progress.Update(msg)
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
	keyMsg, ok := msg.(tea.KeyPressMsg)

	if ok && keyMsg.String() == "ctrl+c" {
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

func isDismissKey(k tea.KeyPressMsg) bool {
	switch k.String() {
	case "enter", "ctrl+c", "q", "esc":
		return true
	default:
		return false
	}
}

func (m *Model) processFormData() {
	m.choices.AddOns = config.NewSet(m.selectedOptions...)

	m.stepMessages = nil
	m.stepMessages = append(m.stepMessages, "Project name: "+m.choices.Name)
	if m.resolved.PythonPath != "" {
		m.stepMessages = append(m.stepMessages, "Python interpreter: "+m.resolved.PythonPath+" ("+m.resolved.PythonVersion+")")
	}
	m.stepMessages = append(m.stepMessages, "Django version: "+m.choices.Django)
	m.stepMessages = append(m.stepMessages, "Database: "+string(m.choices.Database))
	m.stepMessages = append(m.stepMessages, "Stack: "+m.choices.Stack.Spec().Label)
	if m.choices.App != "" {
		m.stepMessages = append(m.stepMessages, "App name: "+m.choices.App)
	}
	m.stepMessages = append(m.stepMessages, addOnSummary(m.selectedOptions, m.choices.AddedAddOns())...)
}

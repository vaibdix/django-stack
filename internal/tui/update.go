package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// Update applies one message to the model.
//
// It runs on Bubble Tea's single goroutine and is the only writer of any Model
// field, which is what makes the model race-free without mutexes. The
// scaffolder reports through a channel that is drained here, rather than
// mutating the model from its own goroutine as it used to.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var spinnerCmd tea.Cmd
	m.Spinner, spinnerCmd = m.Spinner.Update(msg)
	cmds = append(cmds, spinnerCmd)

	if cmd, handled := m.handleExitKeys(msg); handled {
		return m, cmd
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

	// Discovery results. Both arrive asynchronously, so each handler rebuilds
	// its picker and re-Inits it only if the user is already looking at it.
	case pythonOptionsMsg:
		m.applyPythonOptions(msg)
		if m.Step == StepPythonSelection {
			cmds = append(cmds, m.PythonForm.Init())
		}
		return m, tea.Batch(cmds...)

	case djangoOptionsMsg:
		m.applyDjangoOptions(msg.Options, msg.Note)
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

	// The three cases below drain the generator's event channel. Each one
	// re-issues waitForScaffoldEvent so the stream keeps flowing, and each
	// mutates the model here on the main goroutine rather than from the
	// generator's.
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
		m.Progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	return m, tea.Batch(append(cmds, m.updateActiveForm(msg)...)...)
}

// updateActiveForm feeds a message to the form on screen, or advances the flow
// if that form has just been completed.
func (m *Model) updateActiveForm(msg tea.Msg) []tea.Cmd {
	activeForm := m.GetActiveForm()
	if activeForm == nil {
		return nil
	}

	if activeForm.State != huh.StateCompleted {
		formModel, formCmd := activeForm.Update(msg)
		if castedForm, ok := formModel.(*huh.Form); ok {
			m.setActiveForm(castedForm)
		}
		return []tea.Cmd{formCmd}
	}

	return m.advanceFromCompletedForm()
}

// setActiveForm stores the updated form back into the field for the current
// step. huh returns a new value from Update, so it has to be reassigned.
func (m *Model) setActiveForm(form *huh.Form) {
	switch m.Step {
	case StepProjectName:
		m.ProjectNameForm = form
	case StepPythonSelection:
		m.PythonForm = form
	case StepDjangoVersion:
		m.DjangoVersionForm = form
	case StepDatabase:
		m.DatabaseForm = form
	case StepProjectConfig:
		m.ProjectConfigForm = form
	case StepAppName:
		m.AppNameForm = form
	case StepDevServerPrompt:
		m.DevServerForm = form
	}
}

// advanceFromCompletedForm moves to the next screen once the current form is
// done. This is the setup flow's state machine, in one place.
func (m *Model) advanceFromCompletedForm() []tea.Cmd {
	switch m.Step {
	case StepProjectName:
		// Discovery was started by Init and has almost always landed by now; if
		// it has not, the pending picker is on screen and pythonOptionsMsg
		// swaps it out when it arrives.
		m.Step = StepPythonSelection
		return []tea.Cmd{m.PythonForm.Init()}

	case StepPythonSelection:
		m.syncSelectedPythonOption()
		m.Step = StepDjangoVersion
		m.DjangoVersionForm = buildPendingDjangoForm(&m.DjangoVersion)
		return []tea.Cmd{m.fetchDjangoOptionsCmd(), m.DjangoVersionForm.Init()}

	case StepDjangoVersion:
		m.Step = StepDatabase
		return []tea.Cmd{m.DatabaseForm.Init()}

	case StepDatabase:
		m.Step = StepProjectConfig
		return []tea.Cmd{m.ProjectConfigForm.Init()}

	case StepProjectConfig:
		m.ProcessFormData()
		if m.NeedsAppName() {
			m.Step = StepAppName
			return []tea.Cmd{m.AppNameForm.Init()}
		}
		return m.beginGeneration()

	case StepAppName:
		return m.beginGeneration()

	case StepDevServerPrompt:
		if !m.StartDevServer {
			m.Step = StepComplete
			return nil
		}
		m.startDevelopmentEnvironment()
		m.Done = true
		return []tea.Cmd{func() tea.Msg { return tea.KeyMsg{Type: tea.KeyEnter} }}
	}

	return nil
}

// beginGeneration switches to the setup screen and starts the scaffolder.
//
// Reached from two places — with and without an app name — which is why it is a
// method rather than duplicated in both branches.
func (m *Model) beginGeneration() []tea.Cmd {
	m.Step = StepSetup
	m.TotalSteps = m.CalculateTotalSteps()
	m.CompletedSteps = 0
	m.ProgressStatus = "Starting project setup..."

	return []tea.Cmd{
		m.Spinner.Tick,
		m.Progress.SetPercent(0.0),
		m.StartGeneration(),
	}
}

// handleExitKeys deals with the two ways the program ends. The bool reports
// whether the message was consumed.
//
// Quitting mid-scaffold cancels the run first: the context propagates to every
// child process, so the TUI no longer exits while pip keeps writing into a
// half-built project.
func (m *Model) handleExitKeys(msg tea.Msg) (tea.Cmd, bool) {
	keyMsg, ok := msg.(tea.KeyMsg)

	// Mid-run: ctrl-c or q abandons the scaffold.
	if ok && !m.Done && m.Error == nil {
		if keyMsg.Type == tea.KeyCtrlC || keyMsg.String() == "q" {
			m.CancelGeneration()
			return tea.Quit, true
		}
	}

	// Finished or failed: any dismissal key exits, everything else is ignored.
	if m.Error != nil || m.Done {
		if ok && isDismissKey(keyMsg) {
			return tea.Quit, true
		}
		return nil, true
	}

	return nil, false
}

// isDismissKey reports whether k dismisses the final screen.
func isDismissKey(k tea.KeyMsg) bool {
	switch k.Type {
	case tea.KeyEnter, tea.KeyCtrlC:
		return true
	default:
		return k.String() == "q" || k.String() == "esc"
	}
}

// ProcessFormData folds the feature screen's selections into the project config
// and records the resulting choices in the step log.
//
// The mapping itself lives in config.ApplyFeatures, keyed on config.Feature
// rather than on the labels rendered in the menu, so rewording a menu entry
// cannot silently turn a feature off.
func (m *Model) ProcessFormData() {
	m.ApplyFeatures(m.SelectedOptions)

	m.StepMessages = append(m.StepMessages, "Project name: "+m.ProjectName)
	if m.SelectedPythonPath != "" {
		m.StepMessages = append(m.StepMessages, "Python interpreter: "+m.SelectedPythonPath+" ("+m.SelectedPythonVersion+")")
	}
	m.StepMessages = append(m.StepMessages, "Django version: "+m.DjangoVersion)
	m.StepMessages = append(m.StepMessages, "Database: "+string(m.DatabaseType))
	if m.AppName != "" {
		m.StepMessages = append(m.StepMessages, "App name: "+m.AppName)
	}
	m.StepMessages = append(m.StepMessages, fmt.Sprintf("Selected options: %v", m.SelectedOptions))
}

// BlinkCmd schedules the next frame of the mascot's blink animation.
func BlinkCmd() tea.Cmd {
	return tea.Tick(time.Second/2, func(_ time.Time) tea.Msg {
		return BlinkMsg{}
	})
}

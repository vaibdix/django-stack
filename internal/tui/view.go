package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

// View renders the current screen.
//
// It is a pure function of model state: no field is written here. Progress-bar
// width is clamped in Update's WindowSizeMsg handler, which is where it belongs
// — View used to mutate it, so rendering had a side effect.
func (m *Model) View() string {
	viewWidth := m.Width
	if viewWidth <= 0 {
		viewWidth = 100
	}
	contentWidth := viewWidth - 8
	if contentWidth < 70 {
		contentWidth = 70
	}

	baseStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Padding(1, 4)

	var s strings.Builder

	if m.Error != nil {
		errMsg := errorStyle.Render(fmt.Sprintf("ERROR: %s", m.Error.Error()))
		s.WriteString(errMsg + "\n\n")
		s.WriteString("Press Enter or Q to exit.")
		return baseStyle.Render(s.String())
	}

	if m.Done {
		eyeChar := "◒"
		if !m.EyesOpen {
			eyeChar = "-"
		}
		blinkingFace := fmt.Sprintf("%s ◡ %s", eyeChar, eyeChar)

		if m.StartDevServer {
			s.WriteString(titleStyle.Render("Development Environment Started!") + "\n\n")
			s.WriteString("VS Code has been opened in a fresh workspace window.\n")
			s.WriteString("The workspace dev task is starting the project services in integrated terminals.\n\n")
			if m.SetupTailwind {
				s.WriteString("You should see split terminals for Tailwind and Django inside VS Code.\n\n")
			} else {
				s.WriteString("You should see the Django development server terminal inside VS Code.\n\n")
			}
			s.WriteString("   ╭─────╮\n")
			fmt.Fprintf(&s, "   │ %s     brew install happiness  \n", blinkingFace)
			s.WriteString("   ╰─────╯\n")
		} else {
			s.WriteString(titleStyle.Render("Django Project Setup Complete!") + "\n\n")
			s.WriteString(subtitleStyle.Render("What's Next:") + "\n")
			fmt.Fprintf(&s, "1. Navigate to your project directory:\n   cd %s\n\n", m.ProjectName)

			projectAbsPath, _ := filepath.Abs(m.ProjectName)
			pythonVenvPath := pyenv.InterpreterPath(projectAbsPath)
			fmt.Fprintf(&s, "2. Start the development server:\n   %s manage.py runserver\n\n", pythonVenvPath)
		}

		s.WriteString("\n" + footerStyle.Render("Press Enter or Q to exit."))
		return baseStyle.Render(s.String())
	}

	if m.Step == StepSplashScreen {
		djangoDisplayVersion := m.DjangoVersion
		if djangoDisplayVersion == "" || djangoDisplayVersion == "latest" {
			djangoDisplayVersion = "latest stable"
		}
		s.WriteString(titleStyle.Render("Django Stack CLI") + "\n\n")
		fmt.Fprintf(&s, "Welcome! Initializing Django project creator with Django %s.\n", djangoDisplayVersion)
		fmt.Fprintf(&s, "Starting in %d seconds...\n\n", m.SplashCountdown)
		s.WriteString(subtitleStyle.Render("Crafting your Django project, one step at a time."))
		return baseStyle.Render(s.String())
	}

	activeForm := m.GetActiveForm()

	switch m.Step {
	case StepSetup:
		s.WriteString(titleStyle.Render("Project Initialization") + "\n\n")
		fmt.Fprintf(&s, "%s %s\n\n", m.Spinner.View(), m.ProgressStatus)

		// Progress width is clamped in Update's tea.WindowSizeMsg handler.
		// View must stay a pure function of model state.

		pad := strings.Repeat(" ", 2)

		s.WriteString(pad + m.Progress.View() + "\n\n")

		percentage := int(m.Progress.Percent() * 100)
		s.WriteString(pad + progressStyle.Render(fmt.Sprintf("Progress: %d%%", percentage)) + "\n")

		if recent := m.RecentStepMessages(recentStepMessageCount); len(recent) > 0 {
			s.WriteString("\n" + pad + "Recent steps:\n")
			for _, msg := range recent {
				s.WriteString(pad + "  " + msg + "\n")
			}
		}

		return contentBox.Width(contentWidth).Render(s.String())

	case StepProjectName, StepPythonSelection, StepDjangoVersion,
		StepDatabase, StepProjectConfig, StepAppName, StepDevServerPrompt:
		return baseStyle.Render(m.renderFormScreen(activeForm))

	case StepComplete:
		eyeChar := "◒"
		if !m.EyesOpen {
			eyeChar = "-"
		}
		blinkingFace := fmt.Sprintf("%s ◡ %s", eyeChar, eyeChar)

		s.WriteString(titleStyle.Render("All Done!") + "\n\n")
		s.WriteString("   ╭─────╮\n")
		fmt.Fprintf(&s, "   │ %s     Time to code like a wizard! \n", blinkingFace)
		s.WriteString("   ╰─────╯\n")
		s.WriteString("\n\n")

		s.WriteString(subtitleStyle.Render("Manual Steps:") + "\n")
		fmt.Fprintf(&s, "1. Navigate to your project: cd %s\n", m.ProjectName)
		projectAbsPath, _ := filepath.Abs(m.ProjectName)
		pythonVenvPath := pyenv.InterpreterPath(projectAbsPath)
		if m.SetupTailwind {
			s.WriteString("2. Start CSS watching: npm run watch:css\n")
			fmt.Fprintf(&s, "3. Start Django server: %s manage.py runserver\n", pythonVenvPath)
		} else {
			fmt.Fprintf(&s, "2. Start Django server: %s manage.py runserver\n", pythonVenvPath)
		}
		s.WriteString("\n" + footerStyle.Render("Press Enter or Q to exit."))
		return baseStyle.Render(s.String())
	}

	return baseStyle.Render(s.String())
}

// formScreenTitles is the heading shown above each form. The seven form screens
// were seven near-identical switch cases differing only by this string.
var formScreenTitles = map[Step]string{
	StepProjectName:     "Django Project Configuration",
	StepPythonSelection: "Python Selection",
	StepDjangoVersion:   "Django Version Selection",
	StepDatabase:        "Database Selection",
	StepProjectConfig:   "Project Features",
	StepAppName:         "App Configuration",
	StepDevServerPrompt: "Project Setup Complete!",
}

// renderFormScreen renders the heading, any note, and the form itself.
func (m *Model) renderFormScreen(activeForm *huh.Form) string {
	if activeForm == nil {
		return ""
	}

	var s strings.Builder
	s.WriteString(titleStyle.Render(formScreenTitles[m.Step]))

	// The prompt screen wants a blank line under its heading; the pickers do not.
	if m.Step == StepDevServerPrompt {
		s.WriteString("\n\n")
	} else {
		s.WriteString("\n")
	}

	// The version picker explains where its list came from — PyPI or the
	// bundled fallback — which matters when the network is unavailable.
	if m.Step == StepDjangoVersion && m.DjangoVersionNote != "" {
		s.WriteString(m.DjangoVersionNote + "\n\n")
	}

	s.WriteString(activeForm.View())
	return s.String()
}

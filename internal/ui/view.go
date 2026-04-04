package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
		errMsg := ErrorStyle.Render(fmt.Sprintf("ERROR: %s", m.Error.Error()))
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
			s.WriteString(TitleStyle.Render("Development Environment Started!") + "\n\n")
			s.WriteString("VS Code has been opened in a fresh workspace window.\n")
			s.WriteString("The workspace dev task is starting the project services in integrated terminals.\n\n")
			if m.SetupTailwind {
				s.WriteString("You should see split terminals for Tailwind and Django inside VS Code.\n\n")
			} else {
				s.WriteString("You should see the Django development server terminal inside VS Code.\n\n")
			}
			s.WriteString("   ╭─────╮\n")
			s.WriteString(fmt.Sprintf("   │ %s     brew install happiness  \n", blinkingFace))
			s.WriteString("   ╰─────╯\n")
		} else {
			s.WriteString(TitleStyle.Render("Django Project Setup Complete!") + "\n\n")
			s.WriteString(SubtitleStyle.Render("What's Next:") + "\n")
			s.WriteString(fmt.Sprintf("1. Navigate to your project directory:\n   cd %s\n\n", m.ProjectName))

			projectAbsPath, _ := filepath.Abs(m.ProjectName)
			pythonVenvPath := getPythonPath(projectAbsPath)
			s.WriteString(fmt.Sprintf("2. Start the development server:\n   %s manage.py runserver\n\n", pythonVenvPath))
		}

		s.WriteString("\n" + FooterStyle.Render("Press Enter or Q to exit."))
		return baseStyle.Render(s.String())
	}

	if m.Step == StepSplashScreen {
		djangoDisplayVersion := m.DjangoVersion
		if djangoDisplayVersion == "" || djangoDisplayVersion == "latest" {
			djangoDisplayVersion = "latest stable"
		}
		s.WriteString(TitleStyle.Render("Django Stack CLI") + "\n\n")
		s.WriteString(fmt.Sprintf("Welcome! Initializing Django project creator with Django %s.\n", djangoDisplayVersion))
		s.WriteString(fmt.Sprintf("Starting in %d seconds...\n\n", m.SplashCountdown))
		s.WriteString(SubtitleStyle.Render("Crafting your Django project, one step at a time."))
		return baseStyle.Render(s.String())
	}

	activeForm := m.GetActiveForm()

	switch m.Step {
	case StepSetup:
		s.WriteString(TitleStyle.Render("Project Initialization") + "\n\n")
		s.WriteString(fmt.Sprintf("%s %s\n\n", m.Spinner.View(), m.ProgressStatus))

		m.Progress.Width = contentWidth - 8
		if m.Progress.Width > 80 {
			m.Progress.Width = 80
		}
		if m.Progress.Width < 20 {
			m.Progress.Width = 20
		}

		pad := strings.Repeat(" ", 2)

		s.WriteString(pad + m.Progress.View() + "\n\n")

		percentage := int(m.Progress.Percent() * 100)
		s.WriteString(pad + ProgressStyle.Render(fmt.Sprintf("Progress: %d%%", percentage)) + "\n")

		if len(m.StepMessages) > 0 {
			s.WriteString("\n" + pad + "Recent steps:\n")
			start := len(m.StepMessages)
			if start > 3 {
				start = len(m.StepMessages) - 3
			}
			for _, msg := range m.StepMessages[start:] {
				s.WriteString(pad + "  " + msg + "\n")
			}
		}

		return ContentBox.Width(contentWidth).Render(s.String())

	case StepProjectName:
		if activeForm != nil {
			s.WriteString(TitleStyle.Render("Django Project Configuration") + "\n")
			s.WriteString(activeForm.View())
		}
		return baseStyle.Render(s.String())

	case StepPythonSelection:
		if activeForm != nil {
			s.WriteString(TitleStyle.Render("Python Selection") + "\n")
			s.WriteString(activeForm.View())
		}
		return baseStyle.Render(s.String())

	case StepDjangoVersion:
		if activeForm != nil {
			s.WriteString(TitleStyle.Render("Django Version Selection") + "\n")
			if m.DjangoVersionNote != "" {
				s.WriteString(m.DjangoVersionNote + "\n\n")
			}
			s.WriteString(activeForm.View())
		}
		return baseStyle.Render(s.String())

	case StepDatabase:
		if activeForm != nil {
			s.WriteString(TitleStyle.Render("Database Selection") + "\n")
			s.WriteString(activeForm.View())
		}
		return baseStyle.Render(s.String())

	case StepProjectConfig:
		if activeForm != nil {
			s.WriteString(TitleStyle.Render("Project Features") + "\n")
			s.WriteString(activeForm.View())
		}
		return baseStyle.Render(s.String())

	case StepAppName:
		if activeForm != nil {
			s.WriteString(TitleStyle.Render("App Configuration") + "\n")
			s.WriteString(activeForm.View())
		}
		return baseStyle.Render(s.String())

	case StepDevServerPrompt:
		if activeForm != nil {
			s.WriteString(TitleStyle.Render("Project Setup Complete!") + "\n\n")
			s.WriteString(activeForm.View())
		}
		return baseStyle.Render(s.String())

	case StepComplete:
		eyeChar := "◒"
		if !m.EyesOpen {
			eyeChar = "-"
		}
		blinkingFace := fmt.Sprintf("%s ◡ %s", eyeChar, eyeChar)

		s.WriteString(TitleStyle.Render("All Done!") + "\n\n")
		s.WriteString("   ╭─────╮\n")
		s.WriteString(fmt.Sprintf("   │ %s     Time to code like a wizard! \n", blinkingFace))
		s.WriteString("   ╰─────╯\n")
		s.WriteString("\n\n")

		s.WriteString(SubtitleStyle.Render("Manual Steps:") + "\n")
		s.WriteString(fmt.Sprintf("1. Navigate to your project: cd %s\n", m.ProjectName))
		projectAbsPath, _ := filepath.Abs(m.ProjectName)
		pythonVenvPath := getPythonPath(projectAbsPath)
		if m.SetupTailwind {
			s.WriteString("2. Start CSS watching: npm run watch:css\n")
			s.WriteString(fmt.Sprintf("3. Start Django server: %s manage.py runserver\n", pythonVenvPath))
		} else {
			s.WriteString(fmt.Sprintf("2. Start Django server: %s manage.py runserver\n", pythonVenvPath))
		}
		s.WriteString("\n" + FooterStyle.Render("Press Enter or Q to exit."))
		return baseStyle.Render(s.String())
	}

	return baseStyle.Render(s.String())
}

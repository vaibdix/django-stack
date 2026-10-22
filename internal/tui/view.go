package tui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

const recentStepsLabel = "Recent steps:"
const wordmark = "django-stack"

func (m *Model) View() string {
	inner := innerWidth(m.width)

	page := layout{
		Header: m.header(inner),
		Footer: m.footer(inner),
	}

	switch {
	case m.failure != nil:
		page.Body, page.CenterBody = m.errorBody(), true
	case m.done:
		page.Body, page.CenterBody = m.doneBody(), true
	default:
		page.Body = m.stepBody(inner)
		page.CenterBody = m.step == stepSplashScreen || m.step == stepComplete
	}

	return page.render(m.width, m.height)
}

func (m *Model) stepBody(inner int) string {
	switch m.step {
	case stepSplashScreen:
		return m.splashBody(inner)

	case stepProjectName, stepPythonSelection, stepDjangoVersion,
		stepDatabase, stepStack, stepProjectConfig, stepAppName, stepDevServerPrompt:
		return m.formBody(m.currentForm())

	case stepDockerCheck:
		return m.dockerCheckBody()

	case stepSetup:
		return m.setupBody()

	case stepComplete:
		return m.completeBody()
	}

	return ""
}

func (m *Model) header(inner int) string {
	left := accentStyle.Render("◆") + " " + headingStyle.Render(wordmark)

	var right string
	if position, ok := stepPositions[m.step]; ok && m.failure == nil && !m.done {
		right = subtleStyle.Render(fmt.Sprintf("Step %d of %d", position, totalFlowSteps))
	}

	return spread(inner, left, right)
}

func (m *Model) footer(inner int) string {
	text := m.footerCopy()

	return columns(inner,
		sectionHead("▤", "Keys", text.hint)+"\n\n"+pillRow(text.keys...),
		sectionHead("◈", "What's next", text.next),
	)
}

type footerText struct {
	keys []string
	hint string
	next string
}

func (m *Model) footerCopy() footerText {
	if _, ok := errors.AsType[*scaffold.VerificationError](m.failure); ok {
		return footerText{
			keys: []string{"⏎ exit", "q exit"},
			hint: "Nothing is still running.",
			next: "The project is in " + m.project.ProjectName + ". Fix the failing check above, then run it there again.",
		}
	}

	if m.failure != nil {
		return footerText{
			keys: []string{"⏎ exit", "q exit"},
			hint: "Nothing is still running.",
			next: "The scaffold stopped where it failed. Its output above says where.",
		}
	}

	if m.done {
		return footerText{
			keys: []string{"⏎ exit", "q exit"},
			hint: "You can close this now.",
			next: "Your project is ready in " + m.project.ProjectName + ".",
		}
	}

	switch m.step {
	case stepSplashScreen:
		return footerText{
			keys: []string{"any key start", "ctrl+c quit"},
			hint: "Press any key to skip the countdown.",
			next: "Naming your project.",
		}

	case stepProjectName:
		return footerText{
			keys: []string{"type to edit", "⏎ next", "ctrl+c quit"},
			hint: "The arrow keys move the cursor inside the field.",
			next: "Choosing the Python interpreter to build the virtualenv with.",
		}

	case stepPythonSelection:
		return footerText{
			keys: []string{"↑↓ move", "←→ step", "⏎ select", "ctrl+c quit"},
			hint: "← returns to the project name; answers are kept.",
			next: "Choosing a Django release compatible with it.",
		}

	case stepDjangoVersion:
		return footerText{
			keys: []string{"↑↓ move", "←→ step", "⏎ select", "ctrl+c quit"},
			hint: "← returns to the interpreter and refetches this list.",
			next: "Choosing the database.",
		}

	case stepDatabase:
		return footerText{
			keys: []string{"↑↓ move", "←→ step", "⏎ select", "ctrl+c quit"},
			hint: "Docker options generate a compose file and wait for the container.",
			next: "Choosing the stack the project is built from.",
		}

	case stepDockerCheck:
		if m.dockerChecking {
			return footerText{
				keys: []string{"ctrl+c quit"},
				hint: "Asking Docker whether it is running.",
				next: "Starting the database container once Docker answers.",
			}
		}
		return footerText{
			keys: []string{"↑↓ move", "←→ step", "⏎ select", "ctrl+c quit"},
			hint: "Nothing has been written to disk yet, so retrying costs nothing.",
			next: "Choosing the stack the project is built from.",
		}

	case stepStack:
		return footerText{
			keys: []string{"↑↓ move", "←→ step", "⏎ select", "ctrl+c quit"},
			hint: "This decides which HTML templates the project is built from.",
			next: "Choosing the add-ons to wire in.",
		}

	case stepProjectConfig:
		return footerText{
			keys: []string{"↑↓ move", "x toggle", "←→ step", "⏎ confirm", "ctrl+c quit"},
			hint: "Space or x toggles an add-on.",
			next: "Naming the app, if the add-ons you picked need one.",
		}

	case stepAppName:
		return footerText{
			keys: []string{"⏎ next", "⇧⇥ back", "ctrl+c quit"},
			hint: "The arrow keys move the cursor; shift+tab reopens the feature list.",
			next: "Building the project. Nothing else is asked after this.",
		}

	case stepSetup:
		if m.cancelling {
			return footerText{
				keys: []string{"ctrl+c force quit"},
				hint: "Waiting for the run to stop and clean up after itself.",
				next: "The partial project and its database are being removed.",
			}
		}
		return footerText{
			keys: []string{"ctrl+c cancel"},
			hint: "Quitting cancels the run and stops every child process.",
			next: fmt.Sprintf("%d of %d steps done.", m.completedSteps, m.totalSteps),
		}

	case stepDevServerPrompt:
		return footerText{
			keys: []string{"←→ move", "⏎ confirm", "ctrl+c quit"},
			hint: "Yes opens a fresh VS Code workspace and starts the dev task.",
			next: "Everything is scaffolded, linted and formatted.",
		}

	case stepComplete:
		return footerText{
			keys: []string{"⏎ exit", "q exit"},
			hint: "You can close this now.",
			next: "Start the server when you are ready.",
		}
	}

	return footerText{keys: []string{"ctrl+c quit"}}
}

func (m *Model) splashBody(inner int) string {
	version := m.project.DjangoVersion
	if version == "" {
		version = "latest stable"
	}

	var s strings.Builder
	s.WriteString(heroMark(inner) + "\n\n")
	s.WriteString(headingStyle.Render(screenTitles[stepSplashScreen]) + "\n\n")
	s.WriteString(bodyStyle.Render("Welcome! Initializing Django project creator with Django "+version+".") + "\n\n")
	s.WriteString(accentPill(fmt.Sprintf("Starting in %d seconds", m.splashCountdown)) + "\n\n")
	s.WriteString(bodyStyle.Render("Crafting your Django project, one step at a time."))

	return centered(inner, s.String())
}

func (m *Model) formBody(activeForm *huh.Form) string {
	if activeForm == nil {
		return ""
	}

	var s strings.Builder
	s.WriteString(headingStyle.Render(screenTitles[m.step]) + "\n")

	if m.step == stepDjangoVersion && m.djangoVersionNote != "" {
		s.WriteString(bodyStyle.Render(m.djangoVersionNote) + "\n")
	}

	s.WriteString("\n" + activeForm.View())
	return s.String()
}

func (m *Model) dockerCheckBody() string {
	var s strings.Builder
	s.WriteString(headingStyle.Render(screenTitles[stepDockerCheck]) + "\n\n")

	if m.dockerChecking {
		fmt.Fprintf(&s, "%s %s", m.spinner.View(), bodyStyle.Render("Checking whether Docker is running..."))
		return s.String()
	}

	s.WriteString(dangerStyle.Render(m.dockerProblem) + "\n\n")
	s.WriteString(bodyStyle.Render("The project has not been created yet. Start Docker and retry, "+
		"or pick a database that does not need it.") + "\n")

	if form := m.currentForm(); form != nil {
		s.WriteString("\n" + form.View())
	}
	return s.String()
}

func (m *Model) setupBody() string {
	var s strings.Builder
	s.WriteString(headingStyle.Render(screenTitles[stepSetup]) + "\n\n")

	fmt.Fprintf(&s, "%s %s\n\n", m.spinner.View(), bodyStyle.Render(m.progressStatus))

	s.WriteString(m.progress.View() + "\n\n")

	percent := int(m.progress.Percent() * 100)
	s.WriteString(accentPill(fmt.Sprintf("Progress: %d%%", percent)))

	if recent := m.recentStepMessages(recentStepMessageCount); len(recent) > 0 {
		s.WriteString("\n\n" + headingStyle.Render(recentStepsLabel) + "\n")
		for _, msg := range recent {
			s.WriteString(subtleStyle.Render("·") + " " + bodyStyle.Render(msg) + "\n")
		}
	}

	return strings.TrimRight(s.String(), "\n")
}

func (m *Model) completeBody() string {
	var s strings.Builder
	s.WriteString(headingStyle.Render(screenTitles[stepComplete]) + "\n\n")
	s.WriteString(m.mascot("Time to code like a wizard!") + "\n\n")

	s.WriteString(headingStyle.Render("Manual Steps:") + "\n")
	s.WriteString(numbered(1, "Navigate to your project: ", chip("cd "+m.project.ProjectName)) + "\n")

	runserver := chip(m.venvPython() + " manage.py runserver")
	if m.project.SetupTailwind {
		s.WriteString(numbered(2, "Start CSS watching: ", chip(m.watchCSSCommand())) + "\n")
		s.WriteString(numbered(3, "Start Django server: ", runserver))
	} else {
		s.WriteString(numbered(2, "Start Django server: ", runserver))
	}

	return s.String()
}

func (m *Model) doneBody() string {
	var s strings.Builder

	if m.startDevServer {
		s.WriteString(headingStyle.Render("Development Environment Started!") + "\n\n")
		s.WriteString(bodyStyle.Render("VS Code has been opened in a fresh workspace window.") + "\n")
		s.WriteString(bodyStyle.Render("The workspace dev task is starting the project services in integrated terminals.") + "\n\n")
		if m.project.SetupTailwind {
			s.WriteString(bodyStyle.Render("You should see split terminals for Tailwind and Django inside VS Code.") + "\n\n")
		} else {
			s.WriteString(bodyStyle.Render("You should see the Django development server terminal inside VS Code.") + "\n\n")
		}
		s.WriteString(m.mascot("brew install happiness"))
		return s.String()
	}

	s.WriteString(headingStyle.Render("Django Project Setup Complete!") + "\n\n")
	s.WriteString(headingStyle.Render("What's Next:") + "\n")
	s.WriteString(numbered(1, "Navigate to your project directory:", "") + "\n")
	s.WriteString("  " + chip("cd "+m.project.ProjectName) + "\n\n")
	s.WriteString(numbered(2, "Start the development server:", "") + "\n")
	s.WriteString("  " + chip(m.venvPython()+" manage.py runserver"))

	return s.String()
}

func (m *Model) errorBody() string {
	return dangerStyle.Render("ERROR") + "\n\n" + bodyStyle.Render(m.failure.Error())
}

func (m *Model) mascot(message string) string {
	face := accentStyle.Render("◒ ◡ ◒")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 2).
		Render(face + "   " + successStyle.Render(message))
}

func numbered(n int, text, trailing string) string {
	return accentStyle.Render(fmt.Sprintf("%d.", n)) + " " + bodyStyle.Render(text) + trailing
}

func (m *Model) venvPython() string {
	projectAbsPath, _ := filepath.Abs(m.project.ProjectName)
	return pyenv.InterpreterPath(projectAbsPath)
}

func spread(width int, left, right string) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

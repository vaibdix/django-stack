package tui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

const recentStepsLabel = "Recent steps:"
const wordmark = "django-stack"

func (m *Model) View() tea.View {
	view := tea.NewView(m.render())
	view.AltScreen = true
	return view
}

func (m *Model) render() string {
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
	if position, ok := stepPosition(m.step); ok && m.failure == nil && !m.done {
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
			keys: exitKeys,
			hint: "Nothing is still running.",
			next: "The project is in " + m.project.ProjectName + ". Fix the failing check above, then run it there again.",
		}
	}
	if m.failure != nil {
		return footerText{keys: exitKeys, hint: "Nothing is still running.", next: "The scaffold stopped where it failed. Its output above says where."}
	}
	if m.done {
		return footerText{keys: exitKeys, hint: "You can close this now.", next: "Your project is ready in " + m.project.ProjectName + "."}
	}

	spec, ok := steps()[m.step]
	switch {
	case !ok:
		return footerText{keys: []string{"ctrl+c quit"}}
	case (m.dockerChecking || m.cancelling) && spec.busyFooter.keys != nil:
		return spec.busyFooter
	}
	text := spec.footer
	if spec.footerNext != nil {
		text.next = spec.footerNext(m)
	}
	return text
}

func (m *Model) splashBody(inner int) string {
	version := m.project.DjangoVersion
	if version == "" {
		version = "latest stable"
	}

	var s strings.Builder
	s.WriteString(heroMark(inner) + "\n\n")
	s.WriteString(headingStyle.Render(screenTitle(stepSplashScreen)) + "\n\n")
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
	s.WriteString(headingStyle.Render(screenTitle(m.step)) + "\n")

	if m.step == stepDjangoVersion && m.djangoVersionNote != "" {
		s.WriteString(bodyStyle.Render(m.djangoVersionNote) + "\n")
	}

	s.WriteString("\n" + activeForm.View())
	return s.String()
}

func (m *Model) dockerCheckBody() string {
	var s strings.Builder
	s.WriteString(headingStyle.Render(screenTitle(stepDockerCheck)) + "\n\n")

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
	s.WriteString(headingStyle.Render(screenTitle(stepSetup)) + "\n\n")

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
	s.WriteString(headingStyle.Render(screenTitle(stepComplete)) + "\n\n")
	s.WriteString(m.mascot("Time to code like a wizard!") + "\n\n")

	s.WriteString(headingStyle.Render("Manual Steps:") + "\n")
	s.WriteString(numbered(1, "Navigate to your project: ", chip("cd "+m.project.ProjectName)) + "\n")

	runserver := chip(m.venvPython() + " manage.py runserver")
	if m.project.UsesTailwind() {
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
		if m.project.UsesTailwind() {
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

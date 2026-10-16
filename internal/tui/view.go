package tui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/vaibdix/django-cli/internal/scaffold"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

const recentStepsLabel = "Recent steps:"
const wordmark = "django-stack"

var screenTitles = map[Step]string{
	StepSplashScreen:    "Get started",
	StepProjectName:     "Django Project Configuration",
	StepPythonSelection: "Python Selection",
	StepDjangoVersion:   "Django Version Selection",
	StepDatabase:        "Database Selection",
	StepDockerCheck:     "Docker Check",
	StepStack:           "Stack Selection",
	StepProjectConfig:   "Project Features",
	StepAppName:         "App Configuration",
	StepSetup:           "Project Initialization",
	StepDevServerPrompt: "Project Setup Complete!",
	StepComplete:        "All Done!",
}

var stepPositions = map[Step]int{
	StepProjectName:     1,
	StepPythonSelection: 2,
	StepDjangoVersion:   3,
	StepDatabase:        4,
	StepDockerCheck:     4,
	StepStack:           5,
	StepProjectConfig:   6,
	StepAppName:         7,
	StepSetup:           8,
}

const totalFlowSteps = 8

func (m *Model) View() string {
	inner := innerWidth(m.Width)

	page := layout{
		Header: m.header(inner),
		Footer: m.footer(inner),
	}

	switch {
	case m.Error != nil:
		page.Body, page.CenterBody = m.errorBody(), true
	case m.Done:
		page.Body, page.CenterBody = m.doneBody(), true
	default:
		page.Body = m.stepBody(inner)
		page.CenterBody = m.Step == StepSplashScreen || m.Step == StepComplete
	}

	return page.render(m.Width, m.Height)
}

func (m *Model) stepBody(inner int) string {
	switch m.Step {
	case StepSplashScreen:
		return m.splashBody(inner)

	case StepProjectName, StepPythonSelection, StepDjangoVersion,
		StepDatabase, StepStack, StepProjectConfig, StepAppName, StepDevServerPrompt:
		return m.formBody(m.GetActiveForm())

	case StepDockerCheck:
		return m.dockerCheckBody()

	case StepSetup:
		return m.setupBody()

	case StepComplete:
		return m.completeBody()
	}

	return ""
}

func (m *Model) header(inner int) string {
	left := accentStyle.Render("◆") + " " + headingStyle.Render(wordmark)

	var right string
	if position, ok := stepPositions[m.Step]; ok && m.Error == nil && !m.Done {
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
	var verification *scaffold.VerificationError
	if errors.As(m.Error, &verification) {
		return footerText{
			keys: []string{"⏎ exit", "q exit"},
			hint: "Nothing is still running.",
			next: "The project is in " + m.ProjectName + ". Fix the failing check above, then run it there again.",
		}
	}

	if m.Error != nil {
		return footerText{
			keys: []string{"⏎ exit", "q exit"},
			hint: "Nothing is still running.",
			next: "The scaffold stopped where it failed. Its output above says where.",
		}
	}

	if m.Done {
		return footerText{
			keys: []string{"⏎ exit", "q exit"},
			hint: "You can close this now.",
			next: "Your project is ready in " + m.ProjectName + ".",
		}
	}

	switch m.Step {
	case StepSplashScreen:
		return footerText{
			keys: []string{"any key start", "ctrl+c quit"},
			hint: "Press any key to skip the countdown.",
			next: "Naming your project.",
		}

	case StepProjectName:
		return footerText{
			keys: []string{"type to edit", "⏎ next", "ctrl+c quit"},
			hint: "The arrow keys move the cursor inside the field.",
			next: "Choosing the Python interpreter to build the virtualenv with.",
		}

	case StepPythonSelection:
		return footerText{
			keys: []string{"↑↓ move", "←→ step", "⏎ select", "ctrl+c quit"},
			hint: "← returns to the project name; answers are kept.",
			next: "Choosing a Django release compatible with it.",
		}

	case StepDjangoVersion:
		return footerText{
			keys: []string{"↑↓ move", "←→ step", "⏎ select", "ctrl+c quit"},
			hint: "← returns to the interpreter and refetches this list.",
			next: "Choosing the database.",
		}

	case StepDatabase:
		return footerText{
			keys: []string{"↑↓ move", "←→ step", "⏎ select", "ctrl+c quit"},
			hint: "Docker options generate a compose file and wait for the container.",
			next: "Choosing the stack the project is built from.",
		}

	case StepDockerCheck:
		if m.DockerChecking {
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

	case StepStack:
		return footerText{
			keys: []string{"↑↓ move", "←→ step", "⏎ select", "ctrl+c quit"},
			hint: "This decides which HTML templates the project is built from.",
			next: "Choosing the add-ons to wire in.",
		}

	case StepProjectConfig:
		return footerText{
			keys: []string{"↑↓ move", "x toggle", "←→ step", "⏎ confirm", "ctrl+c quit"},
			hint: "Space or x toggles an add-on.",
			next: "Naming the app, if the add-ons you picked need one.",
		}

	case StepAppName:
		return footerText{
			keys: []string{"⏎ next", "⇧⇥ back", "ctrl+c quit"},
			hint: "The arrow keys move the cursor; shift+tab reopens the feature list.",
			next: "Building the project. Nothing else is asked after this.",
		}

	case StepSetup:
		if m.Cancelling {
			return footerText{
				keys: []string{"ctrl+c force quit"},
				hint: "Waiting for the run to stop and clean up after itself.",
				next: "The partial project and its database are being removed.",
			}
		}
		return footerText{
			keys: []string{"ctrl+c cancel"},
			hint: "Quitting cancels the run and stops every child process.",
			next: fmt.Sprintf("%d of %d steps done.", m.CompletedSteps, m.TotalSteps),
		}

	case StepDevServerPrompt:
		return footerText{
			keys: []string{"←→ move", "⏎ confirm", "ctrl+c quit"},
			hint: "Yes opens a fresh VS Code workspace and starts the dev task.",
			next: "Everything is scaffolded, linted and formatted.",
		}

	case StepComplete:
		return footerText{
			keys: []string{"⏎ exit", "q exit"},
			hint: "You can close this now.",
			next: "Start the server when you are ready.",
		}
	}

	return footerText{keys: []string{"ctrl+c quit"}}
}

func (m *Model) splashBody(inner int) string {
	version := m.DjangoVersion
	if version == "" || version == "latest" {
		version = "latest stable"
	}

	var s strings.Builder
	s.WriteString(heroMark(inner) + "\n\n")
	s.WriteString(headingStyle.Render(screenTitles[StepSplashScreen]) + "\n\n")
	s.WriteString(bodyStyle.Render("Welcome! Initializing Django project creator with Django "+version+".") + "\n\n")
	s.WriteString(accentPill(fmt.Sprintf("Starting in %d seconds", m.SplashCountdown)) + "\n\n")
	s.WriteString(bodyStyle.Render("Crafting your Django project, one step at a time."))

	return centered(inner, s.String())
}

func (m *Model) formBody(activeForm *huh.Form) string {
	if activeForm == nil {
		return ""
	}

	var s strings.Builder
	s.WriteString(headingStyle.Render(screenTitles[m.Step]) + "\n")

	if m.Step == StepDjangoVersion && m.DjangoVersionNote != "" {
		s.WriteString(bodyStyle.Render(m.DjangoVersionNote) + "\n")
	}

	s.WriteString("\n" + activeForm.View())
	return s.String()
}

func (m *Model) dockerCheckBody() string {
	var s strings.Builder
	s.WriteString(headingStyle.Render(screenTitles[StepDockerCheck]) + "\n\n")

	if m.DockerChecking {
		fmt.Fprintf(&s, "%s %s", m.Spinner.View(), bodyStyle.Render("Checking whether Docker is running..."))
		return s.String()
	}

	s.WriteString(dangerStyle.Render(m.DockerProblem) + "\n\n")
	s.WriteString(bodyStyle.Render("The project has not been created yet. Start Docker and retry, "+
		"or pick a database that does not need it.") + "\n")

	if form := m.GetActiveForm(); form != nil {
		s.WriteString("\n" + form.View())
	}
	return s.String()
}

func (m *Model) setupBody() string {
	var s strings.Builder
	s.WriteString(headingStyle.Render(screenTitles[StepSetup]) + "\n\n")

	fmt.Fprintf(&s, "%s %s\n\n", m.Spinner.View(), bodyStyle.Render(m.ProgressStatus))

	s.WriteString(m.Progress.View() + "\n\n")

	percent := int(m.Progress.Percent() * 100)
	s.WriteString(accentPill(fmt.Sprintf("Progress: %d%%", percent)))

	if recent := m.RecentStepMessages(recentStepMessageCount); len(recent) > 0 {
		s.WriteString("\n\n" + headingStyle.Render(recentStepsLabel) + "\n")
		for _, msg := range recent {
			s.WriteString(subtleStyle.Render("·") + " " + bodyStyle.Render(msg) + "\n")
		}
	}

	return strings.TrimRight(s.String(), "\n")
}

func (m *Model) completeBody() string {
	var s strings.Builder
	s.WriteString(headingStyle.Render(screenTitles[StepComplete]) + "\n\n")
	s.WriteString(m.mascot("Time to code like a wizard!") + "\n\n")

	s.WriteString(headingStyle.Render("Manual Steps:") + "\n")
	s.WriteString(numbered(1, "Navigate to your project: ", chip("cd "+m.ProjectName)) + "\n")

	runserver := chip(m.venvPython() + " manage.py runserver")
	if m.SetupTailwind {
		s.WriteString(numbered(2, "Start CSS watching: ", chip(m.watchCSSCommand())) + "\n")
		s.WriteString(numbered(3, "Start Django server: ", runserver))
	} else {
		s.WriteString(numbered(2, "Start Django server: ", runserver))
	}

	return s.String()
}

func (m *Model) doneBody() string {
	var s strings.Builder

	if m.StartDevServer {
		s.WriteString(headingStyle.Render("Development Environment Started!") + "\n\n")
		s.WriteString(bodyStyle.Render("VS Code has been opened in a fresh workspace window.") + "\n")
		s.WriteString(bodyStyle.Render("The workspace dev task is starting the project services in integrated terminals.") + "\n\n")
		if m.SetupTailwind {
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
	s.WriteString("  " + chip("cd "+m.ProjectName) + "\n\n")
	s.WriteString(numbered(2, "Start the development server:", "") + "\n")
	s.WriteString("  " + chip(m.venvPython()+" manage.py runserver"))

	return s.String()
}

func (m *Model) errorBody() string {
	return dangerStyle.Render("ERROR") + "\n\n" + bodyStyle.Render(m.Error.Error())
}

func (m *Model) mascot(message string) string {
	eye := "◒"
	if !m.EyesOpen {
		eye = "-"
	}
	face := accentStyle.Render(fmt.Sprintf("%s ◡ %s", eye, eye))

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
	projectAbsPath, _ := filepath.Abs(m.ProjectName)
	return pyenv.InterpreterPath(projectAbsPath)
}

func spread(width int, left, right string) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

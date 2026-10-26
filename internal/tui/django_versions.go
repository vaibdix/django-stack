package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/releases"
)

type djangoOptionsMsg struct {
	Options     []releases.Option
	Note        string
	Interpreter string
	Seq         int
}

func (m *Model) fetchDjangoOptionsCmd() tea.Cmd {
	m.djangoRequest++
	seq := m.djangoRequest
	pythonOption, ok := m.selectedPythonOption()
	if !ok {
		return func() tea.Msg {
			return djangoOptionsMsg{Note: "Choose a Python interpreter first.", Seq: seq}
		}
	}
	today := m.clock()
	catalog, fetch := m.catalog, m.djangoOptions
	return func() tea.Msg {
		options, note := fetch(context.Background(), catalog, pythonOption, today)
		return djangoOptionsMsg{Options: options, Note: note, Interpreter: pythonOption.Path, Seq: seq}
	}
}

func (m *Model) isCurrentDjangoReply(msg djangoOptionsMsg) bool {
	return msg.Seq == m.djangoRequest && (msg.Interpreter == "" || msg.Interpreter == m.resolved.PythonPath)
}

func (m *Model) applyDjangoOptions(options []releases.Option, note string) {
	m.availableDjangoOptions = options
	m.djangoVersionNote = note

	if len(options) == 0 {
		m.choices.Django = ""
		m.djangoVersionForm = buildUnavailableDjangoForm(&m.choices.Django, note)
		return
	}

	if m.choices.Django == "" || !m.hasDjangoOption(m.choices.Django) {
		m.choices.Django = options[0].Version
	}
	m.djangoVersionForm = m.buildDjangoVersionForm()
}

func buildPendingDjangoForm(target *string) *huh.Form {
	return selectForm("Django Version", "Fetching the latest compatible Django releases...",
		[]huh.Option[string]{huh.NewOption("Loading...", "")}, target, requireDjango)
}

func (m *Model) buildDjangoVersionForm() *huh.Form {
	options := make([]huh.Option[string], 0, len(m.availableDjangoOptions))
	for _, option := range m.availableDjangoOptions {
		options = append(options, huh.NewOption(option.Label, option.Version))
	}

	description := "Pick one of the latest compatible stable Django releases."
	if m.djangoVersionNote != "" {
		description = m.djangoVersionNote
	}

	return selectForm("Django Version", description, options, &m.choices.Django, requireDjango)
}

func buildUnavailableDjangoForm(target *string, description string) *huh.Form {
	if description == "" {
		description = "No compatible Django versions are available for the selected Python interpreter."
	}
	if target != nil {
		*target = ""
	}
	return selectForm("Django Version", description,
		[]huh.Option[string]{huh.NewOption("No compatible stable Django release found", "")}, target, requireDjango)
}

func (m *Model) hasDjangoOption(version string) bool {
	for _, option := range m.availableDjangoOptions {
		if option.Version == version {
			return true
		}
	}
	return false
}

func djangoReleaseOptions(ctx context.Context, catalog *releases.Catalog, python pyenv.Option, today time.Time) ([]releases.Option, string) {
	options, err := catalog.Options(ctx, python, today)
	var unsupported *releases.UnsupportedPythonError
	var unreadable *releases.FetchError
	switch {
	case errors.As(err, &unsupported):
		return nil, fmt.Sprintf("No supported Django release runs on Python %d.%d yet; the newest it supports is 3.%d. "+
			"Press ← and pick an older interpreter.", unsupported.Major, unsupported.Minor, unsupported.Newest)
	case errors.As(err, &unreadable):
		return nil, fmt.Sprintf("The Django releases could not be read from %s (%v). "+
			"Check the connection, then press ← and choose the interpreter again to retry.", unreadable.Index, unreadable.Err)
	case err != nil:
		return nil, err.Error()
	case len(options) == 0:
		return nil, fmt.Sprintf("PyPI lists no supported Django release for Python %d.%d.", python.Major, python.Minor)
	}
	return options, fmt.Sprintf("The newest release of each Django series that supports Python %d.%d, from PyPI.",
		python.Major, python.Minor)
}

func requireDjango(version string) error {
	if version == "" {
		return errors.New("no Django release to install for this interpreter; press ← and pick another one")
	}
	return nil
}

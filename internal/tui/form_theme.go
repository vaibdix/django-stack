package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var formTheme = func() *huh.Theme {
	t := huh.ThemeBase()
	t.Focused.Base = t.Focused.Base.BorderForeground(colorBorder)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = headingStyle
	t.Focused.NoteTitle = headingStyle.MarginBottom(1)
	t.Focused.Description = bodyStyle
	t.Focused.Directory = accentStyle
	t.Focused.File = bodyStyle

	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(colorDanger)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(colorDanger)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(colorAccent)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(colorAccent)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(colorAccent)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(colorAccent)

	t.Focused.Option = bodyStyle
	t.Focused.UnselectedOption = bodyStyle
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(colorHeading)
	t.Focused.SelectedPrefix = accentStyle.SetString("✓ ")
	t.Focused.UnselectedPrefix = subtleStyle.SetString("· ")
	t.Focused.FocusedButton = accentPillStyle.Padding(0, 2).MarginRight(1)
	t.Focused.BlurredButton = pillStyle.Padding(0, 2).MarginRight(1)
	t.Focused.Next = t.Focused.FocusedButton

	t.Focused.TextInput.Cursor = accentStyle
	t.Focused.TextInput.Prompt = accentStyle
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(colorHeading)
	t.Focused.TextInput.Placeholder = subtleStyle

	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = headingStyle
	t.Group.Description = bodyStyle

	return t
}()

func newForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(formTheme).
		WithShowHelp(false)
}

func (m *Model) syncFormWidths() {
	width := innerWidth(m.width)
	for _, form := range []*huh.Form{
		m.projectNameForm, m.pythonForm, m.djangoVersionForm, m.databaseForm, m.stackForm,
		m.dockerForm, m.projectConfigForm, m.appNameForm, m.devServerForm,
	} {
		if form != nil {
			form.WithWidth(width)
		}
	}
}

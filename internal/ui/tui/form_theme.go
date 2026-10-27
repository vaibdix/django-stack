package tui

import (
	"math"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

var formTheme = huh.ThemeFunc(func(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)
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
})

func newForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(formTheme).
		WithShowHelp(false)
}

func selectForm[T comparable](title, description string, options []huh.Option[T], target *T, validate func(T) error) *huh.Form {
	field := huh.NewSelect[T]().Title(title).Description(description).Options(options...)
	if validate != nil {
		field = field.Validate(validate)
	}
	return newForm(huh.NewGroup(field.Value(target)))
}

func (m *Model) syncFormWidths() {
	width := innerWidth(m.width)
	for _, spec := range steps() {
		if spec.form == nil {
			continue
		}
		if form := *spec.form(m); form != nil {
			form.WithWidth(width)
		}
	}
	m.fitActiveForm()
}

func (m *Model) fitActiveForm() {
	if form := m.currentForm(); form != nil {
		form.Update(tea.WindowSizeMsg{Width: innerWidth(m.width), Height: math.MaxInt32})
	}
}

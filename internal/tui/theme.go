package tui

import (
	"image/color"
	"sync/atomic"

	"charm.land/lipgloss/v2"
)

var lightBackground atomic.Bool

type adaptiveColor struct{ light, dark color.Color }

func adaptive(light, dark string) adaptiveColor {
	return adaptiveColor{light: lipgloss.Color(light), dark: lipgloss.Color(dark)}
}

func (c adaptiveColor) RGBA() (r, g, b, a uint32) {
	if lightBackground.Load() {
		return c.light.RGBA()
	}
	return c.dark.RGBA()
}

var (
	colorText    = adaptive("#6b6375", "#9ca3af")
	colorHeading = adaptive("#08060d", "#f3f4f6")
	colorBorder  = adaptive("#e5e4e7", "#2e303a")
	colorCodeBG  = adaptive("#f4f3ec", "#1f2028")
	colorAccent  = adaptive("#aa3bff", "#c084fc")

	colorAccentBG = adaptive("#f6ebff", "#2f273e")

	colorSurface = adaptive("#f9f9f5", "#22232b")
	colorDanger  = adaptive("#ff4672", "#ed567a")
	colorSuccess = adaptive("#02ba84", "#02bf87")
)

var (
	headingStyle    = lipgloss.NewStyle().Foreground(colorHeading).Bold(true)
	bodyStyle       = lipgloss.NewStyle().Foreground(colorText)
	subtleStyle     = lipgloss.NewStyle().Foreground(colorBorder)
	accentStyle     = lipgloss.NewStyle().Foreground(colorAccent)
	railStyle       = lipgloss.NewStyle().Foreground(colorBorder)
	chipStyle       = lipgloss.NewStyle().Foreground(colorHeading).Background(colorCodeBG)
	pillStyle       = lipgloss.NewStyle().Foreground(colorHeading).Background(colorSurface)
	accentPillStyle = lipgloss.NewStyle().Foreground(colorAccent).Background(colorAccentBG).Bold(true)
	dangerStyle     = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	successStyle    = lipgloss.NewStyle().Foreground(colorSuccess)
)

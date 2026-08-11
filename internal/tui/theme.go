package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorText    = lipgloss.AdaptiveColor{Light: "#6b6375", Dark: "#9ca3af"}
	colorHeading = lipgloss.AdaptiveColor{Light: "#08060d", Dark: "#f3f4f6"}
	colorBorder  = lipgloss.AdaptiveColor{Light: "#e5e4e7", Dark: "#2e303a"}
	colorCodeBG  = lipgloss.AdaptiveColor{Light: "#f4f3ec", Dark: "#1f2028"}
	colorAccent  = lipgloss.AdaptiveColor{Light: "#aa3bff", Dark: "#c084fc"}

	colorAccentBG = lipgloss.AdaptiveColor{Light: "#f6ebff", Dark: "#2f273e"}

	colorSurface = lipgloss.AdaptiveColor{Light: "#f9f9f5", Dark: "#22232b"}
	colorDanger  = lipgloss.AdaptiveColor{Light: "#ff4672", Dark: "#ed567a"}
	colorSuccess = lipgloss.AdaptiveColor{Light: "#02ba84", Dark: "#02bf87"}
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

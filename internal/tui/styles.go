package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Align(lipgloss.Center)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Align(lipgloss.Center)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Align(lipgloss.Center)

	contentBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))
)

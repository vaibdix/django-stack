package tui

import "github.com/charmbracelet/lipgloss"

// The palette, taken from the reference design's CSS custom properties.
//
// Each colour is an AdaptiveColor so the CLI reads correctly on a light and a
// dark terminal; the Dark values are the reference's dark scheme verbatim.
//
// Two of the reference's properties are declared there as translucent overlays
// rather than as solid colours. A terminal cell has no alpha channel, so those
// two are composited against their backdrop here and stored as the resulting
// solid hex. The arithmetic is written out so the values can be re-derived when
// the reference changes, rather than looking like arbitrary constants.
var (
	colorText    = lipgloss.AdaptiveColor{Light: "#6b6375", Dark: "#9ca3af"}
	colorHeading = lipgloss.AdaptiveColor{Light: "#08060d", Dark: "#f3f4f6"}
	colorBorder  = lipgloss.AdaptiveColor{Light: "#e5e4e7", Dark: "#2e303a"}
	colorCodeBG  = lipgloss.AdaptiveColor{Light: "#f4f3ec", Dark: "#1f2028"}
	colorAccent  = lipgloss.AdaptiveColor{Light: "#aa3bff", Dark: "#c084fc"}

	//	--accent-bg light: rgba(170, 59, 255, .10) over #ffffff
	//	  = (.1*170 + .9*255, .1*59 + .9*255, .1*255 + .9*255) = (246, 235, 255)
	//	--accent-bg dark:  rgba(192, 132, 252, .15) over #16171d
	//	  = (.15*192 + .85*22, .15*132 + .85*23, .15*252 + .85*29) = (47, 39, 62)
	colorAccentBG = lipgloss.AdaptiveColor{Light: "#f6ebff", Dark: "#2f273e"}
	//	--social-bg light: rgba(244, 243, 236, .5) over #ffffff = (249, 249, 245)
	//	--social-bg dark:  rgba(47, 48, 58, .5)   over #16171d = (34, 35, 43)
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

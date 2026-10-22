package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	maxFrameWidth   = 96
	minFrameWidth   = 30
	framePadding    = 2
	stackBreakpoint = 64
)

func frameWidth(termWidth int) int {
	switch {
	case termWidth <= 0:
		return maxFrameWidth
	case termWidth > maxFrameWidth:
		return maxFrameWidth
	default:
		return termWidth
	}
}

func framePad(outer int) int {
	if outer < minFrameWidth {
		return 0
	}
	return framePadding
}

func innerWidth(termWidth int) int { return innerFor(frameWidth(termWidth)) }

func innerFor(outer int) int {
	w := outer - 2 - 2*framePad(outer)
	if w < 1 {
		return 1
	}
	return w
}

func fitLine(s string, w int) string {
	s = lipgloss.NewStyle().MaxWidth(w).Render(s)
	if gap := w - lipgloss.Width(s); gap > 0 {
		s += strings.Repeat(" ", gap)
	}
	return s
}

func railLine(content string, outer, inner int) string {
	rail := railStyle.Render("│")
	pad := strings.Repeat(" ", framePad(outer))
	return rail + pad + fitLine(content, inner) + pad + rail
}

func rule(outer int) string {
	if outer < 2 {
		return ""
	}
	return railStyle.Render("├" + strings.Repeat("─", outer-2) + "┤")
}

func paneLines(body string, outer int) []string {
	inner := innerFor(outer)
	wrapped := lipgloss.NewStyle().Width(inner).Render(body)

	lines := strings.Split(wrapped, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, railLine(line, outer, inner))
	}
	return out
}

func blankLines(n, outer int) []string {
	out := make([]string, 0, n)
	for range n {
		out = append(out, railLine("", outer, innerFor(outer)))
	}
	return out
}

type layout struct {
	Header     string
	Body       string
	Footer     string
	CenterBody bool
}

func (l layout) render(termWidth, termHeight int) string {
	outer := frameWidth(termWidth)
	divider := rule(outer)

	var head []string
	if l.Header != "" {
		head = append(head, paneLines(l.Header, outer)...)
		head = append(head, divider)
	}

	var foot []string
	if l.Footer != "" {
		foot = append(foot, divider)
		foot = append(foot, paneLines(l.Footer, outer)...)
	}
	foot = append(foot, divider, railLine("", outer, innerFor(outer)))

	body := paneLines(l.Body, outer)

	rows := make([]string, 0, len(head)+len(body)+len(foot))
	rows = append(rows, head...)
	slack := termHeight - len(head) - len(foot) - len(body)
	if termHeight <= 0 || slack <= 0 {
		rows = append(rows, body...)
	} else {
		above := 1
		if l.CenterBody {
			above = slack / 2
		}
		rows = append(rows, blankLines(above, outer)...)
		rows = append(rows, body...)
		rows = append(rows, blankLines(slack-above, outer)...)
	}
	rows = append(rows, foot...)

	if margin := (termWidth - outer) / 2; margin > 0 {
		indent := strings.Repeat(" ", margin)
		for i, row := range rows {
			rows[i] = indent + row
		}
	}

	return strings.Join(rows, "\n")
}

func columns(inner int, left, right string) string {
	if inner < stackBreakpoint {
		return left + "\n\n" + right
	}

	const gap = 3
	leftWidth := (inner - gap) / 2
	rightWidth := inner - gap - leftWidth

	leftBlock := lipgloss.NewStyle().Width(leftWidth).MaxWidth(leftWidth).Render(left)
	rightBlock := lipgloss.NewStyle().Width(rightWidth).MaxWidth(rightWidth).Render(right)

	height := max(lipgloss.Height(leftBlock), lipgloss.Height(rightBlock))
	dividerRows := make([]string, height)
	for i := range dividerRows {
		dividerRows[i] = " " + railStyle.Render("│") + " "
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, strings.Join(dividerRows, "\n"), rightBlock)
}

func chip(text string) string { return chipStyle.Render(" " + text + " ") }

func pill(label string) string { return pillStyle.Render(" " + label + " ") }

func accentPill(label string) string { return accentPillStyle.Render(" " + label + " ") }

func pillRow(labels ...string) string {
	rendered := make([]string, 0, len(labels))
	for _, label := range labels {
		rendered = append(rendered, pill(label))
	}
	return strings.Join(rendered, " ")
}

func sectionHead(icon, title, subtitle string) string {
	return accentStyle.Render(icon) + "\n" +
		headingStyle.Render(title) + "\n" +
		bodyStyle.Render(subtitle)
}

func centered(inner int, block string) string {
	return lipgloss.NewStyle().Width(inner).Align(lipgloss.Center).Render(block)
}

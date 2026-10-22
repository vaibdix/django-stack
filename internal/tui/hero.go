package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type heroSpan struct {
	text  string
	style lipgloss.Style
}

const heroMinWidth = 26

var heroWide = [][]heroSpan{
	{{"   ╭" + strings.Repeat("─", 17) + "╮", railStyle}},
	{{"   │" + strings.Repeat(" ", 17) + "│", railStyle}},
	{{"╭──┴" + strings.Repeat("─", 14) + "╮", accentStyle}, {"  │", railStyle}},
	{{"│" + strings.Repeat(" ", 17) + "│", accentStyle}, {"  │", railStyle}},
	{{"│" + strings.Repeat(" ", 17) + "│", accentStyle}, {"  │", railStyle}},
	{{"│" + strings.Repeat(" ", 17) + "├", accentStyle}, {"──╯", railStyle}},
	{{"│" + strings.Repeat(" ", 17) + "│", accentStyle}},
	{{"╰" + strings.Repeat("─", 17) + "╯", accentStyle}},
}

var heroNarrow = [][]heroSpan{
	{{"  ╭─────╮", railStyle}},
	{{"╭─┴───╮", accentStyle}, {" │", railStyle}},
	{{"│     │", accentStyle}, {" │", railStyle}},
	{{"│     ├", accentStyle}, {"─╯", railStyle}},
	{{"╰─────╯", accentStyle}},
}

func heroMark(inner int) string {
	rows := heroWide
	if inner < heroMinWidth {
		rows = heroNarrow
	}

	lines := make([]string, len(rows))
	widths := make([]int, len(rows))
	widest := 0

	for i, row := range rows {
		var line strings.Builder
		for _, span := range row {
			line.WriteString(span.style.Render(span.text))
			widths[i] += lipgloss.Width(span.text)
		}
		lines[i] = line.String()
		widest = max(widest, widths[i])
	}

	for i := range lines {
		lines[i] += strings.Repeat(" ", widest-widths[i])
	}
	return strings.Join(lines, "\n")
}

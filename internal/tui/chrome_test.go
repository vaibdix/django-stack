package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

var testWidths = []int{20, 40, 64, 80, 96, 100, 200}

func TestFrameNeverExceedsTheTerminalWidth(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("a line of body copy that is comfortably long. ", 6)

	for _, width := range testWidths {
		page := layout{
			Header: spread(innerWidth(width), "◆ django-stack", "Step 3 of 7"),
			Body:   body,
			Footer: columns(innerWidth(width), sectionHead("▤", "Keys", "hint"), sectionHead("◈", "Next", "next")),
		}

		for i, line := range strings.Split(page.render(width, 30), "\n") {
			if got := lipgloss.Width(line); got > width {
				t.Errorf("at terminal width %d, line %d is %d cells wide:\n%q",
					width, i, got, line)
			}
		}
	}
}

func TestFrameRowsAreAllTheSameWidth(t *testing.T) {
	t.Parallel()

	for _, width := range testWidths {
		page := layout{
			Header: "header",
			Body:   "body\nwith\nseveral\nlines",
			Footer: "footer",
		}

		lines := strings.Split(page.render(width, 24), "\n")
		want := lipgloss.Width(lines[0])
		for i, line := range lines {
			if got := lipgloss.Width(line); got != want {
				t.Errorf("at terminal width %d, row 0 is %d cells but row %d is %d:\n%q",
					width, want, i, got, line)
			}
		}
	}
}

func TestRuleIsExactlyTheFrameWidth(t *testing.T) {
	t.Parallel()

	for _, width := range testWidths {
		outer := frameWidth(width)
		if got := lipgloss.Width(rule(outer)); got != outer {
			t.Errorf("rule for frame width %d is %d cells wide, want %d", outer, got, outer)
		}
	}
}

func TestInnerWidthIsAlwaysPositive(t *testing.T) {
	t.Parallel()

	for width := -5; width <= 220; width++ {
		if got := innerWidth(width); got < 1 {
			t.Fatalf("innerWidth(%d) = %d, want at least 1", width, got)
		}
	}
}

func TestFrameStopsGrowingAtTheMaximumWidth(t *testing.T) {
	t.Parallel()

	if got := frameWidth(200); got != maxFrameWidth {
		t.Errorf("frameWidth(200) = %d, want the column to stop at %d", got, maxFrameWidth)
	}
	if got := frameWidth(80); got != 80 {
		t.Errorf("frameWidth(80) = %d, want the column to fill a terminal narrower than the maximum", got)
	}

	first := strings.Split(layout{Body: "x"}.render(200, 0), "\n")[0]
	if !strings.HasPrefix(first, " ") {
		t.Errorf("at 200 columns the frame is not indented, so it is not centred:\n%q", first)
	}
}

func TestColumnsStackOnNarrowTerminals(t *testing.T) {
	t.Parallel()

	const left, right = "LEFT", "RIGHT"

	stacked := columns(stackBreakpoint-1, left, right)
	for _, line := range strings.Split(stacked, "\n") {
		if strings.Contains(line, left) && strings.Contains(line, right) {
			t.Errorf("below the breakpoint both halves share a line, so they did not stack:\n%q", line)
		}
	}

	side := columns(stackBreakpoint, left, right)
	if !strings.Contains(strings.Split(side, "\n")[0], left) ||
		!strings.Contains(strings.Split(side, "\n")[0], right) {
		t.Errorf("at the breakpoint the halves should sit side by side:\n%s", side)
	}
}

func TestFitLinePadsAndTruncatesToExactly(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"", "short", strings.Repeat("wide ", 40), "unicode ─┴├ ✓"} {
		for _, width := range []int{1, 10, 40} {
			if got := lipgloss.Width(fitLine(in, width)); got != width {
				t.Errorf("fitLine(%q, %d) is %d cells wide, want exactly %d", in, width, got, width)
			}
		}
	}
}

func TestHeroMarkIsRectangular(t *testing.T) {
	t.Parallel()

	for _, inner := range []int{10, heroMinWidth - 1, heroMinWidth, 90} {
		lines := strings.Split(heroMark(inner), "\n")
		want := lipgloss.Width(lines[0])
		for i, line := range lines {
			if got := lipgloss.Width(line); got != want {
				t.Errorf("at content width %d, hero row 0 is %d cells but row %d is %d:\n%q",
					inner, want, i, got, line)
			}
		}
	}
}

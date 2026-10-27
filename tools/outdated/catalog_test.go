package main

import (
	"strings"
	"testing"
	"time"
)

const djangoTable = `[[django]]
series = "6.1"
python = [12, 14]
supported_through = "2027-12"

[[django]]
series = "5.2"
python = [10, 14]
lts = true
supported_through = "2028-03"

[tools]
tailwind = "4.1.0"
`

func TestAPinIsRewrittenInPlace(t *testing.T) {
	t.Parallel()

	got := rewritePin(djangoTable, `(?m)^tailwind = "([^"]+)"`, "4.2.1")
	if !strings.Contains(got, `tailwind = "4.2.1"`) || strings.Count(got, "\n") != strings.Count(djangoTable, "\n") {
		t.Errorf("rewritePin:\n%s", got)
	}
}

func TestTheDjangoTableFollowsTheReleaseSchedule(t *testing.T) {
	t.Parallel()

	schedule := []release{
		{Cycle: "6.2", EOL: "2028-08-31", ReleaseDate: "2027-04-01"},
		{Cycle: "7.0", EOL: "2029-04-30", ReleaseDate: "2027-12-01"},
		{Cycle: "6.1", EOL: "2027-12-31", ReleaseDate: "2026-08-05"},
		{Cycle: "5.2", EOL: "2028-04-30", LTS: true, ReleaseDate: "2025-04-02"},
		{Cycle: "5.1", EOL: "2025-12-03", ReleaseDate: "2024-08-07"},
	}
	got, changes := updateDjango(djangoTable, schedule, time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC))
	want := `[[django]]
series = "6.2"
python = [12, 14]
supported_through = "2028-08"

[[django]]
series = "6.1"
python = [12, 14]
supported_through = "2027-12"

[[django]]
series = "5.2"
python = [10, 14]
lts = true
supported_through = "2028-04"
`
	if !strings.HasPrefix(got, want) {
		t.Errorf("table:\n%s\nwant it to start with:\n%s", got, want)
	}
	if len(changes) != 2 || !strings.Contains(changes[0], "5.2 supported through 2028-03 -> 2028-04") || !strings.Contains(changes[1], "6.2 added") {
		t.Errorf("changes = %q", changes)
	}
}

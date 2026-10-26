package scaffold

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

const atEnd = ""

type entry struct {
	name  string
	after string
	rank  int
}

func placeEntries(kind string, base []string, entries []entry) ([]string, error) {
	seen := map[string]bool{}
	for _, name := range base {
		seen[name] = true
	}
	slots := map[string]map[int]string{}
	byAnchor := map[string][]entry{}
	for _, e := range entries {
		if seen[e.name] {
			return nil, fmt.Errorf("%s %s is listed twice", kind, e.name)
		}
		seen[e.name] = true
		if slots[e.after] == nil {
			slots[e.after] = map[int]string{}
		}
		if other, taken := slots[e.after][e.rank]; taken {
			return nil, fmt.Errorf("%s %s and %s both claim place %d after %q", kind, other, e.name, e.rank, e.after)
		}
		slots[e.after][e.rank] = e.name
		byAnchor[e.after] = append(byAnchor[e.after], e)
	}
	for anchor := range byAnchor {
		if anchor != atEnd && !seen[anchor] {
			return nil, fmt.Errorf("%s %s goes after %s, which is not in the list", kind, byAnchor[anchor][0].name, anchor)
		}
	}
	for _, group := range byAnchor {
		slices.SortFunc(group, func(a, b entry) int { return cmp.Compare(a.rank, b.rank) })
	}

	var placed []string
	var follow func(name string)
	follow = func(name string) {
		placed = append(placed, name)
		for _, e := range byAnchor[name] {
			follow(e.name)
		}
	}
	for _, name := range base {
		follow(name)
	}
	for _, e := range byAnchor[atEnd] {
		follow(e.name)
	}
	return placed, nil
}

type section int

const (
	beforeUserModel section = iota
	afterUserModel
)

type block struct {
	section section
	rank    int
	text    string
}

func orderBlocks(kind string, blocks []block) ([]block, error) {
	sorted := slices.Clone(blocks)
	slices.SortStableFunc(sorted, func(a, b block) int {
		return cmp.Or(cmp.Compare(a.section, b.section), cmp.Compare(a.rank, b.rank))
	})
	for i := 1; i < len(sorted); i++ {
		if sorted[i].section == sorted[i-1].section && sorted[i].rank == sorted[i-1].rank {
			return nil, fmt.Errorf("two %s blocks claim place %d:\n%s\n%s", kind, sorted[i].rank,
				strings.TrimSpace(sorted[i-1].text), strings.TrimSpace(sorted[i].text))
		}
	}
	return sorted, nil
}

type route struct {
	rank int
	line string
}

func orderRoutes(routes []route) ([]string, error) {
	sorted := slices.Clone(routes)
	slices.SortStableFunc(sorted, func(a, b route) int { return cmp.Compare(a.rank, b.rank) })
	lines := make([]string, len(sorted))
	for i, r := range sorted {
		if i > 0 && r.rank == sorted[i-1].rank {
			return nil, fmt.Errorf("two URL routes claim place %d:\n%s\n%s", r.rank, sorted[i-1].line, r.line)
		}
		lines[i] = r.line
	}
	return lines, nil
}

const (
	lastDjangoApp        = "django.contrib.staticfiles"
	lastDjangoMiddleware = "django.middleware.clickjacking.XFrameOptionsMiddleware"
)

const (
	rankPasswords = iota + 1
	rankLogin
	rankCSPReport
	rankEnvironment
	rankConsoleLogging
)

const (
	rankCSP       = 10
	rankHTMX      = 20
	rankAllAuth   = 30
	rankStructlog = 40
	rankSentry    = 50
	rankMypy      = 60
	rankAPI       = 70
)

const (
	rankHomeRoute    = 10
	rankAdminRoute   = 20
	rankHealthRoute  = 30
	rankAuthRoute    = 40
	rankAllAuthRoute = 41
	rankAPIRoute     = 50
	rankAppRoute     = 60
)

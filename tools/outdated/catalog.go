package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/version"
)

type release struct {
	Cycle       string `json:"cycle"`
	EOL         string `json:"eol"`
	LTS         bool   `json:"lts"`
	ReleaseDate string `json:"releaseDate"`
}

func updateCatalog(ctx context.Context, root string, holds []string, today time.Time) ([]string, error) {
	path := filepath.Join(root, catalogFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(raw)
	var changes []string

	pins, err := pinsIn(root)
	if err != nil {
		return nil, err
	}
	for _, p := range pins {
		if p.file != catalogFile || strings.Contains(p.pattern, "image") || slices.Contains(holds, p.name) {
			continue
		}
		pinned, err := readPin(root, p)
		if err != nil {
			return nil, err
		}
		latest, err := p.latest(ctx, pinned)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.name, err)
		}
		if !p.behind(pinned, latest) {
			continue
		}
		text = rewritePin(text, p.pattern, latest)
		changes = append(changes, fmt.Sprintf("%s %s -> %s", p.name, pinned, latest))
	}

	var schedule []release
	if err := getJSON(ctx, "https://endoflife.date/api/django.json", &schedule); err != nil {
		return nil, fmt.Errorf("the Django release schedule: %w", err)
	}
	text, djangoChanges := updateDjango(text, schedule, today)
	changes = append(changes, djangoChanges...)

	if text != string(raw) {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			return nil, err
		}
	}
	return changes, nil
}

func rewritePin(text, pattern, latest string) string {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatchIndex(text)
	if m == nil {
		return text
	}
	return text[:m[2]] + latest + text[m[3]:]
}

func compareSeries(a, b string) int {
	left, _ := version.Parse(a)
	right, _ := version.Parse(b)
	return version.Compare(left, right)
}

func updateDjango(text string, schedule []release, today time.Time) (string, []string) {
	var changes []string
	djangoBlock := regexp.MustCompile(`(?m)^\[\[django\]\]\nseries = "([^"]+)"\npython = \[(\d+), (\d+)\]\n(lts = true\n)?supported_through = "([^"]+)"\n`)
	blocks := djangoBlock.FindAllStringSubmatch(text, -1)
	known := map[string]bool{}
	newest := ""
	for _, b := range blocks {
		known[b[1]] = true
		if newest == "" || compareSeries(b[1], newest) > 0 {
			newest = b[1]
		}
	}
	for _, r := range schedule {
		through := monthOf(r.EOL)
		if !known[r.Cycle] {
			continue
		}
		block := regexp.MustCompile(`(?m)(^\[\[django\]\]\nseries = "` + regexp.QuoteMeta(r.Cycle) + `"\npython = \[\d+, \d+\]\n(?:lts = true\n)?supported_through = ")([^"]+)(")`)
		m := block.FindStringSubmatch(text)
		if m == nil || m[2] == through {
			continue
		}
		text = block.ReplaceAllString(text, "${1}"+through+"${3}")
		changes = append(changes, fmt.Sprintf("Django %s supported through %s -> %s", r.Cycle, m[2], through))
	}

	var added []release
	for _, r := range schedule {
		released, err := time.Parse(time.DateOnly, r.ReleaseDate)
		if known[r.Cycle] || newest == "" || compareSeries(r.Cycle, newest) <= 0 || err != nil || released.After(today) {
			continue
		}
		added = append(added, r)
	}
	slices.SortFunc(added, func(a, b release) int { return compareSeries(b.Cycle, a.Cycle) })
	if len(added) > 0 {
		var newestBlock []string
		for _, b := range blocks {
			if b[1] == newest {
				newestBlock = b
			}
		}
		var b strings.Builder
		for _, r := range added {
			fmt.Fprintf(&b, "[[django]]\nseries = %q\npython = [%s, %s]\n", r.Cycle, newestBlock[2], newestBlock[3])
			if r.LTS {
				b.WriteString("lts = true\n")
			}
			fmt.Fprintf(&b, "supported_through = %q\n\n", monthOf(r.EOL))
			changes = append(changes, fmt.Sprintf("Django %s added with Python 3.%s to 3.%s copied from %s; check its release notes", r.Cycle, newestBlock[2], newestBlock[3], newest))
		}
		first := strings.Index(text, "[[django]]")
		text = text[:first] + b.String() + text[first:]
	}
	return text, changes
}

func monthOf(date string) string {
	if len(date) >= len("2006-01") {
		return date[:len("2006-01")]
	}
	return date
}

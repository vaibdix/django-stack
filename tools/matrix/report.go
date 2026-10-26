package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type projectReport struct {
	Name     string          `json:"name"`
	Group    string          `json:"group"`
	Generate *generateResult `json:"generate,omitempty"`
	Checks   []checkResult   `json:"checks,omitempty"`
	Diff     *diffResult     `json:"diff,omitempty"`
}

func (p projectReport) failures() []string {
	var out []string
	if p.Generate != nil && !p.Generate.OK {
		out = append(out, fmt.Sprintf("generate (exit %d)", p.Generate.Exit))
	}
	for _, c := range p.Checks {
		if !c.OK && !c.Skipped && !c.Info {
			out = append(out, c.Name)
		}
	}
	return out
}

func runReport(_ context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	dir := fs.String("run", "", "run folder written by select")
	if err := fs.Parse(args); err != nil {
		return err
	}
	r, err := openRun(*dir)
	if err != nil {
		return err
	}
	reports := make([]projectReport, 0, len(r.matrix.Projects))
	for _, p := range r.matrix.Projects {
		pr := projectReport{Name: p.Name, Group: p.Group}
		var gen generateResult
		if readJSON(filepath.Join(r.logDir(p.Name), "generate.json"), &gen) == nil {
			pr.Generate = &gen
		}
		_ = readJSON(filepath.Join(r.logDir(p.Name), "checks.json"), &pr.Checks)
		var d diffResult
		if readJSON(filepath.Join(r.logDir(p.Name), "diff.json"), &d) == nil {
			pr.Diff = &d
		}
		reports = append(reports, pr)
	}
	if err := writeJSON(filepath.Join(r.dir, "results.json"), reports); err != nil {
		return err
	}
	summary := summarize(r, reports)
	if err := os.WriteFile(filepath.Join(r.dir, "summary.md"), []byte(summary), 0o644); err != nil {
		return err
	}
	_, _ = io.WriteString(stdout, summary)
	return nil
}

type tally struct {
	generated, generateOK, checked, checkOK, races, differing, diffed int
	seconds                                                           []float64
}

func (t *tally) add(p projectReport) {
	if g := p.Generate; g != nil {
		t.generated++
		t.generateOK += boolCount(g.OK)
		t.races += boolCount(g.Race)
		if g.Exit == 0 {
			t.seconds = append(t.seconds, g.Seconds)
		}
	}
	if len(p.Checks) > 0 && !p.Checks[0].Skipped {
		t.checked++
		t.checkOK += boolCount(len(p.failures()) == 0)
	}
	if p.Diff != nil {
		t.diffed++
		t.differing += boolCount(!p.Diff.empty())
	}
}

func boolCount(b bool) int {
	if b {
		return 1
	}
	return 0
}

func summarize(r run, reports []projectReport) string {
	var b strings.Builder
	var t tally
	for _, p := range reports {
		t.add(p)
	}
	slices.Sort(t.seconds)
	generated, generateOK, checked, checkOK, races, differing, diffed, seconds := t.generated, t.generateOK, t.checked, t.checkOK, t.races, t.differing, t.diffed, t.seconds

	fmt.Fprintf(&b, "# Sweep %s\n\n", filepath.Base(r.dir))
	fmt.Fprintf(&b, "- Commit: `%s`\n- Selection: %d projects, UV_EXCLUDE_NEWER %s, PICT seed %d\n", r.matrix.Commit, len(r.matrix.Projects), r.matrix.ExcludeNewer, r.matrix.PictSeed)
	fmt.Fprintf(&b, "- Django: %s\n", joinMap(r.matrix.Django))
	fmt.Fprintf(&b, "- Generated as expected: %d of %d; data races: %d\n", generateOK, generated, races)
	if len(seconds) > 0 {
		fmt.Fprintf(&b, "- Generation time (successful runs): median %.1fs, max %.1fs\n", seconds[len(seconds)/2], seconds[len(seconds)-1])
	}
	fmt.Fprintf(&b, "- Checks passed: %d of %d checked projects\n", checkOK, checked)
	if diffed > 0 {
		fmt.Fprintf(&b, "- Differ from the baseline: %d of %d\n", differing, diffed)
	}

	fmt.Fprintf(&b, "\n## Failures\n\n")
	listed := false
	for _, p := range reports {
		if f := p.failures(); len(f) > 0 {
			listed = true
			fmt.Fprintf(&b, "- **%s** (%s): %s\n", p.Name, p.Group, strings.Join(f, ", "))
		}
	}
	if !listed {
		fmt.Fprintf(&b, "None.\n")
	}

	if differing > 0 {
		fmt.Fprintf(&b, "\n## Differences from the baseline\n\n")
		for _, p := range reports {
			if p.Diff != nil && !p.Diff.empty() {
				fmt.Fprintf(&b, "- **%s**: changed %s; only baseline %s; only this run %s\n", p.Name,
					list(p.Diff.Changed), list(p.Diff.OnlyBaseline), list(p.Diff.OnlyRun))
			}
		}
	}
	return b.String()
}

func list(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return "`" + strings.Join(items, "`, `") + "`"
}

func joinMap(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + m[k]
	}
	return strings.Join(parts, ", ")
}

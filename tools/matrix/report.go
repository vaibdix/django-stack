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

func summarize(r run, reports []projectReport) string {
	var b strings.Builder
	generated, generateOK, checked, checkOK, races, differing, diffed := 0, 0, 0, 0, 0, 0, 0
	var seconds []float64
	for _, p := range reports {
		if p.Generate != nil {
			generated++
			if p.Generate.OK {
				generateOK++
			}
			if p.Generate.Race {
				races++
			}
			if p.Generate.Exit == 0 {
				seconds = append(seconds, p.Generate.Seconds)
			}
		}
		if len(p.Checks) > 0 && !p.Checks[0].Skipped {
			checked++
			if len(p.failures()) == 0 {
				checkOK++
			}
		}
		if p.Diff != nil {
			diffed++
			if !p.Diff.empty() {
				differing++
			}
		}
	}
	slices.Sort(seconds)

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

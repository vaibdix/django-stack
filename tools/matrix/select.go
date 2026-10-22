package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/version"
)

const (
	modelFile      = "tools/matrix/model.pict"
	edgeDjango     = "6.1"
	edgePython     = 13
	edgeApp        = "catalog"
	removedOnPy310 = "aifc asynchat asyncore audioop binhex cgi cgitb chunk crypt distutils imghdr imp lib2to3 mailcap msilib nis nntplib ossaudiodev pipes smtpd sndhdr spwd sunau telnetlib uu xdrlib"
)

var (
	boolAddOns = []struct{ column, name string }{
		{"git", "git"}, {"allauth", "allauth"}, {"ruff", "ruff"}, {"mypy", "mypy"},
		{"precommit", "pre-commit"}, {"multienv", "multi-env"}, {"pydantic", "pydantic"},
		{"structlog", "structlog"}, {"sentry", "sentry"},
	}
	apiAddOns = []string{"rest-framework", "django-ninja"}
)

type selection struct {
	django  map[string]string
	pythons map[string]string
	out     []project
}

func runSelect(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("select", flag.ContinueOnError)
	dir := fs.String("run", "", "run folder (default ~/django-stack-runs/<label>-<sha>)")
	label := fs.String("label", "sweep", "run folder label, e.g. baseline or phase0")
	seed := fs.Int("seed", 1, "PICT random seed")
	like := fs.String("like", "", "reuse the selection, dates and versions of an earlier run folder")
	excludeNewer := fs.String("exclude-newer", "", "UV_EXCLUDE_NEWER for every install (default: now)")
	pythons := fs.String("python", "", "interpreters as 3.10=/path,3.11=/path (default: uv python find)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *dir == "" {
		d, err := defaultRunDir(ctx, *label)
		if err != nil {
			return err
		}
		*dir = d
	}
	if err := refuseInsideRepository(ctx, *dir); err != nil {
		return err
	}
	commit, err := gitOutput(ctx, ".", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("read the current commit: %w", err)
	}

	m := matrix{Commit: commit, Created: time.Now().UTC().Truncate(time.Second)}
	if *like != "" {
		earlier, err := openRun(*like)
		if err != nil {
			return err
		}
		m.ExcludeNewer, m.PictSeed = earlier.matrix.ExcludeNewer, earlier.matrix.PictSeed
		m.Django, m.Pythons, m.Projects = earlier.matrix.Django, earlier.matrix.Pythons, earlier.matrix.Projects
	} else {
		m.PictSeed = *seed
		m.ExcludeNewer = *excludeNewer
		if m.ExcludeNewer == "" {
			m.ExcludeNewer = m.Created.Format(time.RFC3339)
		}
		cutoff, err := time.Parse(time.RFC3339, m.ExcludeNewer)
		if err != nil {
			return fmt.Errorf("-exclude-newer must be RFC 3339: %w", err)
		}
		sel, err := buildSelection(ctx, cutoff, *seed, *pythons)
		if err != nil {
			return err
		}
		m.Django, m.Pythons, m.Projects = sel.django, sel.pythons, sel.out
	}

	r := run{dir: *dir, matrix: m}
	for _, p := range m.Projects {
		if err := writeJSON(r.answersPath(p.Name), p.Answers); err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(*dir, matrixFile), m); err != nil {
		return err
	}
	groups := map[string]int{}
	for _, p := range m.Projects {
		groups[p.Group]++
	}
	_, _ = fmt.Fprintf(stdout, "%s: %d projects, UV_EXCLUDE_NEWER=%s\n", *dir, len(m.Projects), m.ExcludeNewer)
	for _, g := range slices.Sorted(maps.Keys(groups)) {
		_, _ = fmt.Fprintf(stdout, "  %-24s %d\n", g, groups[g])
	}
	return nil
}

func buildSelection(ctx context.Context, cutoff time.Time, seed int, pythonFlag string) (selection, error) {
	s := selection{django: map[string]string{}}
	var err error
	if s.pythons, err = findPythons(ctx, pythonFlag); err != nil {
		return s, err
	}
	for _, rule := range config.SupportedDjangoSeries(cutoff) {
		v, err := latestDjango(ctx, rule.Series, cutoff)
		if err != nil {
			return s, err
		}
		s.django[rule.Series] = v
	}

	rows, err := pictRows(ctx, seed)
	if err != nil {
		return s, err
	}
	for i, row := range rows {
		minor, err := strconv.Atoi(row["python"])
		if err != nil {
			return s, fmt.Errorf("pict row %d: python %q", i+1, row["python"])
		}
		var addOns []string
		if row["api"] != "none" {
			addOns = append(addOns, row["api"])
		}
		for _, b := range boolAddOns {
			if row[b.column] == "1" {
				addOns = append(addOns, b.name)
			}
		}
		if err := s.add(fmt.Sprintf("pw%03d", i+1), "3-wise", row["toolchain"], row["node"], expectMade,
			config.Stack(row["stack"]), config.Database(row["database"]), row["django"], minor, addOns); err != nil {
			return s, err
		}
	}
	return s, s.addEdges()
}

func (s *selection) add(name, group, toolchain, node, expect string, stack config.Stack, db config.Database, series string, minor int, addOns []string) error {
	djangoVersion, ok := s.django[series]
	if !ok {
		return fmt.Errorf("%s: Django %s is not supported on the cutoff date", name, series)
	}
	python, ok := s.pythons["3."+strconv.Itoa(minor)]
	if !ok {
		return fmt.Errorf("%s: no Python 3.%d interpreter; install it with uv python install 3.%d", name, minor, minor)
	}
	if !config.DjangoRunsOn(djangoVersion, 3, minor, time.Now()) {
		return fmt.Errorf("%s: Django %s does not run on Python 3.%d; tools/matrix/model.pict disagrees with the support table", name, djangoVersion, minor)
	}
	answers := config.Answers{
		Name: name, Python: python, Django: djangoVersion, Database: db, Stack: stack,
		App: edgeApp, AddOns: append([]string{}, addOns...),
	}
	if expect == expectMade {
		p, err := answers.Project()
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if !p.NeedsAppName() {
			answers.App = ""
		}
	} else {
		answers.App = ""
	}
	s.out = append(s.out, project{Name: name, Group: group, Toolchain: toolchain, Node: node, Expect: expect, Answers: answers})
	return nil
}

func (s *selection) addEdges() error {
	allBools := make([]string, 0, len(boolAddOns))
	for _, b := range boolAddOns {
		allBools = append(allBools, b.name)
	}
	type edge struct {
		name, group, toolchain, node string
		stack                        config.Stack
		db                           config.Database
		series                       string
		minor                        int
		addOns                       []string
	}
	base := func(name, group string, stack config.Stack, addOns ...string) edge {
		return edge{name, group, "uv", "pnpm", stack, config.SQLite, edgeDjango, edgePython, addOns}
	}
	var edges []edge
	for _, stack := range config.AllStacks() {
		edges = append(edges, base("bare-"+string(stack), "bare stack", stack))
	}
	for _, b := range boolAddOns {
		edges = append(edges, base("only-"+b.name, "one add-on", config.StackStandard, b.name))
	}
	for _, api := range apiAddOns {
		edges = append(edges, base("only-"+api, "one add-on", config.StackStandard, api))
		edges = append(edges, base("api-"+api, "one add-on", config.StackAPIOnly, api))
	}
	for _, rule := range config.SupportedDjangoSeries(time.Now()) {
		for minor := rule.MinMinor; minor <= rule.MaxMinor; minor++ {
			e := base(fmt.Sprintf("dj%s-py3%d", strings.ReplaceAll(rule.Series, ".", ""), minor), "django x python", config.StackStandard, "ruff", "mypy")
			e.series, e.minor = rule.Series, minor
			edges = append(edges, e)
		}
	}
	for _, stack := range config.AllStacks() {
		for _, api := range apiAddOns {
			edges = append(edges, base("full-"+string(stack)+"-"+api, "everything on", stack, append([]string{api}, allBools...)...))
		}
	}
	for _, db := range []config.Database{config.PostgresDocker, config.MySQLDocker} {
		e := base("full-"+strings.TrimSuffix(string(db), "-docker"), "everything on", config.StackTailwindHTMX, append([]string{"rest-framework"}, allBools...)...)
		e.db = db
		edges = append(edges, e)
	}
	pipTW := base("full-pip-twhx", "pip only", config.StackTailwindHTMX, append([]string{"rest-framework"}, allBools...)...)
	pipTW.toolchain = "pip"
	pipAPI := base("full-pip-api", "pip only", config.StackAPIOnly, append([]string{"django-ninja"}, allBools...)...)
	pipAPI.toolchain = "pip"
	npmTW := base("npm-tailwind", "npm only", config.StackTailwind, "ruff", "mypy")
	npmTW.node = "npm"
	npmTWHX := base("npm-twhx", "npm only", config.StackTailwindHTMX, "rest-framework", "allauth", "ruff", "mypy")
	npmTWHX.node = "npm"
	n2 := base("n2-precommit-pip", "N2 pre-commit only", config.StackAPIOnly, "pre-commit")
	n2.toolchain = "pip"
	edges = append(edges, pipTW, pipAPI, npmTW, npmTWHX, n2)

	for _, e := range edges {
		if err := s.add(e.name, e.group, e.toolchain, e.node, expectMade, e.stack, e.db, e.series, e.minor, e.addOns); err != nil {
			return err
		}
	}
	for module := range strings.FieldsSeq(removedOnPy310) {
		if err := s.add(module, "N3 stdlib name", "uv", "pnpm", expectNoDir, config.StackAPIOnly, config.SQLite, "5.2", 10, nil); err != nil {
			return err
		}
	}
	return nil
}

func pictRows(ctx context.Context, seed int) ([]map[string]string, error) {
	out, err := exec.CommandContext(ctx, "pict", modelFile, "/o:3", "/r:"+strconv.Itoa(seed)).Output()
	if err != nil {
		if exit, ok := errors.AsType[*exec.ExitError](err); ok {
			return nil, fmt.Errorf("pict: %w\n%s", err, exit.Stderr)
		}
		return nil, fmt.Errorf("pict is required (brew install pict): %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return nil, errors.New("pict produced no rows")
	}
	header := strings.Split(lines[0], "\t")
	rows := make([]map[string]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		fields := strings.Split(line, "\t")
		if len(fields) != len(header) {
			return nil, fmt.Errorf("pict row has %d fields, want %d: %q", len(fields), len(header), line)
		}
		row := make(map[string]string, len(header))
		for i, h := range header {
			row[h] = fields[i]
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func findPythons(ctx context.Context, flagValue string) (map[string]string, error) {
	pythons := map[string]string{}
	if flagValue != "" {
		for pair := range strings.SplitSeq(flagValue, ",") {
			v, path, ok := strings.Cut(pair, "=")
			if !ok {
				return nil, fmt.Errorf("-python wants 3.x=/path pairs, got %q", pair)
			}
			pythons[v] = path
		}
		return pythons, nil
	}
	for minor := 10; minor <= 14; minor++ {
		v := "3." + strconv.Itoa(minor)
		out, err := exec.CommandContext(ctx, "uv", "python", "find", "--no-project", v).Output()
		if err != nil {
			continue
		}
		pythons[v] = strings.TrimSpace(string(out))
	}
	return pythons, nil
}

func latestDjango(ctx context.Context, series string, cutoff time.Time) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://pypi.org/pypi/Django/json", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch Django releases: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var index struct {
		Releases map[string][]struct {
			Uploaded time.Time `json:"upload_time_iso_8601"`
			Yanked   bool      `json:"yanked"`
		} `json:"releases"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&index); err != nil {
		return "", fmt.Errorf("decode Django releases: %w", err)
	}
	prefix, err := version.Parse(series)
	if err != nil {
		return "", err
	}
	var best []int
	bestText := ""
	for v, files := range index.Releases {
		parts, err := version.Parse(v)
		if err != nil || !config.IsStableDjangoVersion(v) || !version.HasPrefix(parts, prefix) || len(files) == 0 {
			continue
		}
		if files[0].Yanked || files[0].Uploaded.After(cutoff) {
			continue
		}
		if best == nil || version.Compare(parts, best) > 0 {
			best, bestText = parts, v
		}
	}
	if bestText == "" {
		return "", fmt.Errorf("no Django %s release before %s", series, cutoff.Format(time.RFC3339))
	}
	return bestText, nil
}

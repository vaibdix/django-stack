package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.yaml.in/yaml/v3"
	"golang.org/x/sync/errgroup"

	"github.com/vaibdix/django-stack/internal/blueprint"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

const tier1Dir = "tier1"

type configuration struct {
	choices  config.Choices
	resolved config.Resolved
	tools    scaffold.Tools
	minor    int
}

type uniqueFile struct {
	Hash    string `json:"hash"`
	Path    string `json:"path"`
	Example string `json:"example"`
	Count   int    `json:"count"`
	Minor   int    `json:"minor"`
	content []byte
}

type interpreter struct {
	series string
	minor  int
}

func interpreters(today time.Time) []interpreter {
	var all []interpreter
	for _, rule := range config.SupportedDjangoSeries(today) {
		for minor := rule.MinMinor; minor <= rule.MaxMinor; minor++ {
			all = append(all, interpreter{rule.Series, minor})
		}
	}
	return all
}

func stackNodes() map[config.Stack][]string {
	nodes := map[config.Stack][]string{}
	for _, stack := range config.AllStacks() {
		nodes[stack] = []string{""}
		if stack.Spec().Tailwind {
			nodes[stack] = []string{"pnpm", "npm"}
		}
	}
	return nodes
}

func addOnSets() []config.Set[config.AddOn] {
	apis := []config.AddOn{"", config.RestFramework, config.Ninja}
	gits := [][]config.AddOn{nil, {config.Git}, {config.Git, config.PreCommit}}
	toggles := []config.AddOn{config.AllAuth, config.Ruff, config.Mypy, config.MultiEnv, config.Pydantic, config.Structlog, config.Sentry}
	var sets []config.Set[config.AddOn]
	for _, api := range apis {
		for _, git := range gits {
			for mask := range 1 << len(toggles) {
				set := config.NewSet(git...)
				if api != "" {
					set = set.With(api)
				}
				for i, toggle := range toggles {
					if mask&(1<<i) != 0 {
						set = set.With(toggle)
					}
				}
				sets = append(sets, set)
			}
		}
	}
	return sets
}

func allConfigurations(today time.Time) []configuration {
	var all []configuration
	nodes, sets := stackNodes(), addOnSets()
	for _, py := range interpreters(today) {
		for _, stack := range config.AllStacks() {
			for _, node := range nodes[stack] {
				manager, _ := scaffold.NodeManagerNamed(node)
				for _, db := range config.Databases() {
					for _, toolchain := range []pyenv.Toolchain{pyenv.UV, pyenv.Pip} {
						for _, addOns := range sets {
							all = append(all, configuration{
								choices: config.Choices{
									Name: "tier-one", Django: py.series + ".0", Database: db,
									Stack: stack, AddOns: addOns, App: edgeApp,
								},
								resolved: config.Resolved{
									PythonVersion: fmt.Sprintf("Python 3.%d.0", py.minor),
									SecretKey:     "tier-one-secret-key",
									Database:      config.DatabaseCredentials{Port: "5432", Password: "tieronepassword", RootPassword: "tieroneroot"},
								},
								tools: scaffold.Tools{Toolchain: toolchain, NodeManager: manager},
								minor: py.minor,
							})
						}
					}
				}
			}
		}
	}
	return all
}

func (c configuration) name() string {
	names := make([]string, 0, len(c.choices.AddOns))
	for _, a := range config.AddOns() {
		if c.choices.AddOns.Has(a) {
			names = append(names, string(a))
		}
	}
	return strings.Join([]string{
		c.choices.Django, strings.TrimPrefix(c.resolved.PythonVersion, "Python "), string(c.choices.Stack),
		string(c.choices.Database), c.tools.Toolchain.String(), c.tools.NodeManager.Name, strings.Join(names, "+"),
	}, "/")
}

func runRenderAll(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("render-all", flag.ContinueOnError)
	dir := fs.String("run", "", "run folder (default ~/django-stack-runs/tier1-<sha>)")
	check := fs.Bool("check", true, "run the G9 checks on every unique file")
	fingerprintFile := fs.String("fingerprint", "", "write one blueprint hash per configuration to this file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		d, err := defaultRunDir(ctx, tier1Dir)
		if err != nil {
			return err
		}
		*dir = d
	}
	if err := refuseInsideRepository(*dir); err != nil {
		return err
	}

	today := time.Now()
	configs := allConfigurations(today)
	unique, prints, err := renderUnique(ctx, configs, today)
	if err != nil {
		return err
	}
	if *fingerprintFile != "" {
		if err := os.WriteFile(*fingerprintFile, fingerprintLines(prints), 0o644); err != nil {
			return err
		}
	}
	root := filepath.Join(*dir, tier1Dir)
	if err := writeUnique(root, unique); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "%s: %d configurations, %d unique files\n", root, len(configs), len(unique))
	if !*check {
		return nil
	}
	return checkUnique(ctx, root, unique, stdout)
}

func renderUnique(ctx context.Context, configs []configuration, today time.Time) (map[string]*uniqueFile, map[string]string, error) {
	var mu sync.Mutex
	unique := map[string]*uniqueFile{}
	prints := map[string]string{}
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(runtime.GOMAXPROCS(0))
	for _, c := range configs {
		group.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			spec, err := config.Parse(c.choices, config.Interpreter{Major: 3, Minor: c.minor}, today)
			if err != nil {
				return fmt.Errorf("%s: %w", c.name(), err)
			}
			bp, err := scaffold.Build(spec, c.resolved, c.tools)
			if err != nil {
				return fmt.Errorf("%s: %w", c.name(), err)
			}
			mu.Lock()
			defer mu.Unlock()
			prints[c.name()] = fingerprint(bp)
			for p, file := range bp.Files {
				sum := sha256.Sum256(append([]byte(path.Base(p)+"\x00"), file.Content...))
				hash := hex.EncodeToString(sum[:])
				if u, ok := unique[hash]; ok {
					u.Count++
					u.Minor = min(u.Minor, c.minor)
					continue
				}
				unique[hash] = &uniqueFile{Hash: hash, Path: p, Example: c.name(), Count: 1, Minor: c.minor, content: file.Content}
			}
			return nil
		})
	}
	err := group.Wait()
	return unique, prints, err
}

func fingerprint(bp blueprint.Blueprint) string {
	h := sha256.New()
	for _, p := range bp.Paths() {
		file := bp.Files[p]
		_, _ = fmt.Fprintf(h, "%s\x00%o\x00%d\x00", p, file.Mode, len(file.Content))
		_, _ = h.Write(file.Content)
	}
	for _, c := range bp.Commands {
		_, _ = fmt.Fprintf(h, "%s\x00%q\x00", c.Name, c.Args)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func fingerprintLines(prints map[string]string) []byte {
	var out bytes.Buffer
	for _, name := range slices.Sorted(maps.Keys(prints)) {
		_, _ = fmt.Fprintf(&out, "%s %s\n", name, prints[name])
	}
	sum := sha256.Sum256(out.Bytes())
	_, _ = fmt.Fprintf(&out, "digest %s\n", hex.EncodeToString(sum[:]))
	return out.Bytes()
}

func (u *uniqueFile) location(root string) string {
	return filepath.Join(root, "files", u.Hash[:16], filepath.FromSlash(u.Path))
}

func writeUnique(root string, unique map[string]*uniqueFile) error {
	index := make([]*uniqueFile, 0, len(unique))
	for _, hash := range slices.Sorted(maps.Keys(unique)) {
		u := unique[hash]
		if err := os.MkdirAll(filepath.Dir(u.location(root)), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(u.location(root), u.content, 0o644); err != nil {
			return err
		}
		index = append(index, u)
	}
	return writeJSON(filepath.Join(root, "index.json"), index)
}

func checkUnique(ctx context.Context, root string, unique map[string]*uniqueFile, stdout io.Writer) error {
	byExt := map[string][]string{}
	pythonByMinor := map[int][]string{}
	var failures []error
	for _, hash := range slices.Sorted(maps.Keys(unique)) {
		u := unique[hash]
		file := u.location(root)
		switch ext := path.Ext(u.Path); ext {
		case ".json":
			if !json.Valid(u.content) {
				failures = append(failures, fmt.Errorf("%s (%s): not valid JSON", u.Path, u.Example))
			}
		case ".yml", ".yaml":
			var doc any
			if err := yaml.Unmarshal(u.content, &doc); err != nil {
				failures = append(failures, fmt.Errorf("%s (%s): %w", u.Path, u.Example, err))
			}
			if path.Base(u.Path) == "docker-compose.yml" {
				byExt["compose"] = append(byExt["compose"], file)
			}
		case ".py":
			byExt[ext] = append(byExt[ext], file)
			pythonByMinor[u.Minor] = append(pythonByMinor[u.Minor], file)
		default:
			byExt[ext] = append(byExt[ext], file)
		}
	}

	checks := []check{
		{"py_compile", byExt[".py"], []string{"python3", "-m", "py_compile"}},
		{"djlint", byExt[".html"], []string{"uvx", "--quiet", "--from", djlintSpec, "djlint", "--lint", "--check", "--profile=django"}},
	}
	for _, minor := range slices.Sorted(maps.Keys(pythonByMinor)) {
		checks = append(checks, ruffChecks(minor, pythonByMinor[minor])...)
	}
	for _, c := range checks {
		if len(c.files) == 0 {
			continue
		}
		if err := runBatched(ctx, c.args, c.files); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", c.name, err))
		}
	}
	for _, compose := range byExt["compose"] {
		cmd := exec.CommandContext(ctx, "docker", "compose", "-f", compose, "config", "-q")
		cmd.Env = append(os.Environ(), "DB_NAME=tier_one", "DB_USER=tier_one", "DB_PASSWORD=tieronepassword", "DB_ROOT_PASSWORD=tieroneroot", "DB_PORT=5432")
		if output, err := cmd.CombinedOutput(); err != nil {
			failures = append(failures, fmt.Errorf("docker compose config %s: %w\n%s", compose, err, output))
		}
	}

	_, _ = fmt.Fprintf(stdout, "G9: %d python, %d templates, %d compose files checked, %d failures\n",
		len(byExt[".py"]), len(byExt[".html"]), len(byExt["compose"]), len(failures))
	return errors.Join(failures...)
}

type check struct {
	name  string
	files []string
	args  []string
}

func ruffChecks(minor int, files []string) []check {
	ruff := []string{"uvx", "--quiet", "ruff"}
	target := []string{"--isolated", "--no-cache", "--target-version", fmt.Sprintf("py3%d", minor)}
	label := fmt.Sprintf(" (Python 3.%d)", minor)
	return []check{
		{"ruff check" + label, files, slices.Concat(ruff, []string{"check"}, target, []string{"--select", "E9,F63,F7,F82"})},
		{"ruff format" + label, files, slices.Concat(ruff, []string{"format", "--check"}, target, []string{"--line-length", "120"})},
		{"ruff imports" + label, slices.DeleteFunc(slices.Clone(files), isMigration), slices.Concat(ruff, []string{"check"}, target, []string{"--select", "I,F401,UP",
			"--config", "lint.isort.force-single-line=true", "--config", `lint.isort.known-first-party=["tier_one","core","accounts","` + edgeApp + `"]`})},
	}
}

func isMigration(file string) bool {
	return filepath.Base(filepath.Dir(file)) == "migrations"
}

const batchSize = 200

func runBatched(ctx context.Context, args, files []string) error {
	var failures []error
	for start := 0; start < len(files); start += batchSize {
		batch := files[start:min(start+batchSize, len(files))]
		cmd := exec.CommandContext(ctx, args[0], append(slices.Clone(args[1:]), batch...)...)
		var output bytes.Buffer
		cmd.Stdout, cmd.Stderr = &output, &output
		if err := cmd.Run(); err != nil {
			failures = append(failures, fmt.Errorf("batch %s: %w\n%s", strconv.Itoa(start/batchSize+1), err, lastLines(output.String(), 40)))
		}
	}
	return errors.Join(failures...)
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return strings.Join(lines[max(0, len(lines)-n):], "\n")
}

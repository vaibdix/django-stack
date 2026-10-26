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

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

const tier1Dir = "tier1"

type configuration struct {
	choices  config.Choices
	resolved config.Resolved
	tools    scaffold.Tools
}

type uniqueFile struct {
	Hash    string `json:"hash"`
	Path    string `json:"path"`
	Example string `json:"example"`
	Count   int    `json:"count"`
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

	configs := allConfigurations(time.Now())
	unique, err := renderUnique(ctx, configs)
	if err != nil {
		return err
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

func renderUnique(ctx context.Context, configs []configuration) (map[string]*uniqueFile, error) {
	var mu sync.Mutex
	unique := map[string]*uniqueFile{}
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(runtime.GOMAXPROCS(0))
	for _, c := range configs {
		group.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			bp, err := scaffold.Build(c.choices, c.resolved, c.tools)
			if err != nil {
				return fmt.Errorf("%s: %w", c.name(), err)
			}
			mu.Lock()
			defer mu.Unlock()
			for p, file := range bp.Files {
				sum := sha256.Sum256(append([]byte(path.Base(p)+"\x00"), file.Content...))
				hash := hex.EncodeToString(sum[:])
				if u, ok := unique[hash]; ok {
					u.Count++
					continue
				}
				unique[hash] = &uniqueFile{Hash: hash, Path: p, Example: c.name(), Count: 1, content: file.Content}
			}
			return nil
		})
	}
	return unique, group.Wait()
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
		default:
			byExt[ext] = append(byExt[ext], file)
		}
	}

	checks := []struct {
		name  string
		files []string
		args  []string
	}{
		{"py_compile", byExt[".py"], []string{"python3", "-m", "py_compile"}},
		{"ruff check", byExt[".py"], []string{"uvx", "--quiet", "ruff", "check", "--isolated", "--no-cache", "--select", "E9,F63,F7,F82"}},
		{"ruff format", byExt[".py"], []string{"uvx", "--quiet", "ruff", "format", "--check", "--isolated", "--no-cache", "--line-length", "120"}},
		{"ruff imports", byExt[".py"], []string{"uvx", "--quiet", "ruff", "check", "--isolated", "--no-cache", "--select", "I,F401,UP",
			"--config", "lint.isort.force-single-line=true", "--config", `lint.isort.known-first-party=["tier_one","core","accounts","` + edgeApp + `"]`}},
		{"djlint", byExt[".html"], []string{"uvx", "--quiet", "--from", djlintSpec, "djlint", "--lint", "--profile=django"}},
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
		if output, err := cmd.CombinedOutput(); err != nil {
			failures = append(failures, fmt.Errorf("docker compose config %s: %w\n%s", compose, err, output))
		}
	}

	_, _ = fmt.Fprintf(stdout, "G9: %d python, %d templates, %d compose files checked, %d failures\n",
		len(byExt[".py"]), len(byExt[".html"]), len(byExt["compose"]), len(failures))
	return errors.Join(failures...)
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

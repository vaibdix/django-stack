package render

import (
	"encoding/json"
	"fmt"
	"maps"
	"math/rand/v2"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"

	"github.com/vaibdix/django-stack/internal/blueprint"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

const specCount = 300

type randomSpec struct {
	spec     config.Spec
	resolved config.Resolved
	tools    Tools
}

func randomName(r *rand.Rand, separators string) string {
	const first, rest = "abcdefghijklmnopqrstuvwxyz", "abcdefghijklmnopqrstuvwxyz0123456789"
	name := []byte{first[r.IntN(len(first))]}
	for range r.IntN(30) {
		if separators != "" && r.IntN(6) == 0 {
			name = append(name, separators[r.IntN(len(separators))])
		}
		name = append(name, rest[r.IntN(len(rest))])
	}
	return string(name)
}

func pick[T any](r *rand.Rand, from []T) T { return from[r.IntN(len(from))] }

func randomSpecs(t *testing.T, seed uint64, want int) []randomSpec {
	t.Helper()
	r := rand.New(rand.NewPCG(seed, seed))
	today := reviewClock()
	var specs []randomSpec
	for tries := 0; len(specs) < want && tries < want*20; tries++ {
		series := pick(r, config.SupportedDjangoSeries(today))
		minor := series.MinMinor + r.IntN(series.MaxMinor-series.MinMinor+1)
		cfg := config.Choices{
			Name:     randomName(r, "-_"),
			App:      randomName(r, "_"),
			Django:   series.Series + ".0",
			Database: pick(r, config.Databases()),
			Stack:    pick(r, config.AllStacks()),
		}
		var addOns []config.AddOn
		for _, addOn := range config.AddOns() {
			if r.IntN(2) == 0 {
				addOns = append(addOns, addOn)
			}
		}
		cfg.AddOns = config.NewSet(addOns...)
		spec, _, err := config.Parse(cfg, config.Interpreter{Major: 3, Minor: minor}, today)
		if err != nil {
			continue
		}
		resolved := credentialsFor(cfg.Database, config.DatabaseCredentials{})
		resolved.PythonVersion = fmt.Sprintf("Python 3.%d.0", minor)
		resolved.SecretKey = randomName(r, "")
		tools := Tools{Toolchain: pick(r, []pyenv.Toolchain{pyenv.UV, pyenv.Pip})}
		if cfg.Stack.Spec().Tailwind {
			tools.NodeManager = pick(r, []NodeManager{Pnpm, NPM})
		}
		specs = append(specs, randomSpec{spec, resolved, tools})
	}
	if len(specs) < want {
		t.Fatalf("only %d of %d random choices parsed; the generator no longer matches the rules", len(specs), want)
	}
	return specs
}

func TestEveryRandomValidSpecRendersWellFormedFiles(t *testing.T) {
	t.Parallel()

	for i, s := range randomSpecs(t, 2026, specCount) {
		label := fmt.Sprintf("%s/%s/%s/%s/%v", s.spec.Name(), s.spec.App(), s.spec.Stack(), s.spec.Database(), s.spec.EnabledAddOns())
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			bp, err := Build(s.spec, s.resolved, s.tools)
			if err != nil {
				t.Fatalf("%s: Build: %v", label, err)
			}
			again, err := Build(s.spec, s.resolved, s.tools)
			if err != nil || !blueprintsEqual(bp, again) {
				t.Errorf("%s: two builds of the same spec differ", label)
			}
			for _, problem := range wellFormed(bp) {
				t.Errorf("%s: %s", label, problem)
			}
		})
	}
}

func blueprintsEqual(a, b blueprint.Blueprint) bool {
	return maps.EqualFunc(a.Files, b.Files, func(x, y blueprint.File) bool {
		return x.Mode == y.Mode && string(x.Content) == string(y.Content)
	})
}

var (
	templateResidue = regexp.MustCompile(`<no value>|%!\(|\[\[[-.]|-\]\]`)
	pythonImport    = regexp.MustCompile(`(?m)^\s*(?:from\s+(\.*[\w.]*)\s+import|import\s+([\w.]+))`)
)

func wellFormed(bp blueprint.Blueprint) []string {
	var problems []string
	for _, name := range bp.Paths() {
		content := string(bp.Files[name].Content)
		if content != "" && !strings.HasSuffix(content, "\n") {
			problems = append(problems, name+" does not end with a newline")
		}
		for line := range strings.SplitSeq(content, "\n") {
			if templateResidue.MatchString(line) {
				problems = append(problems, fmt.Sprintf("%s carries template residue: %q", name, line))
			}
		}
		if err := parses(name, bp.Files[name].Content); err != nil {
			problems = append(problems, fmt.Sprintf("%s does not parse: %v", name, err))
		}
		if path.Ext(name) == ".py" {
			problems = append(problems, unresolvedImports(bp, name, content)...)
		}
	}
	return problems
}

func parses(name string, content []byte) error {
	var doc any
	switch path.Ext(name) {
	case ".json":
		return json.Unmarshal(content, &doc)
	case ".yml", ".yaml":
		return yaml.Unmarshal(content, &doc)
	case ".toml":
		return toml.Unmarshal(content, &doc)
	}
	return nil
}

func unresolvedImports(bp blueprint.Blueprint, name, content string) []string {
	var problems []string
	for _, match := range pythonImport.FindAllStringSubmatch(content, -1) {
		module := match[1] + match[2]
		file, firstParty := modulePath(bp, name, module)
		if !firstParty {
			continue
		}
		if _, ok := bp.Files[file+".py"]; ok {
			continue
		}
		if _, ok := bp.Files[file+"/__init__.py"]; ok {
			continue
		}
		problems = append(problems, fmt.Sprintf("%s imports %s, which the project does not contain", name, module))
	}
	return problems
}

func modulePath(bp blueprint.Blueprint, importer, module string) (string, bool) {
	dots := len(module) - len(strings.TrimLeft(module, "."))
	rest := strings.ReplaceAll(module[dots:], ".", "/")
	if dots == 0 {
		top, _, _ := strings.Cut(rest, "/")
		_, isPackage := bp.Files[top+"/__init__.py"]
		return rest, isPackage
	}
	dir := path.Dir(importer)
	for range dots - 1 {
		dir = path.Dir(dir)
	}
	return strings.TrimSuffix(path.Join(dir, rest), "/."), true
}

func TestTheRandomSpecsCoverEveryChoice(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, s := range randomSpecs(t, 2026, specCount) {
		seen["stack "+string(s.spec.Stack())] = true
		seen["database "+string(s.spec.Database())] = true
		seen["toolchain "+s.tools.Toolchain.String()] = true
		for _, addOn := range s.spec.EnabledAddOns() {
			seen["add-on "+string(addOn)] = true
		}
	}
	var missing []string
	for _, stack := range config.AllStacks() {
		missing = appendIfUnseen(missing, seen, "stack "+string(stack))
	}
	for _, db := range config.Databases() {
		missing = appendIfUnseen(missing, seen, "database "+string(db))
	}
	for _, addOn := range config.AddOns() {
		missing = appendIfUnseen(missing, seen, "add-on "+string(addOn))
	}
	if len(missing) > 0 {
		t.Errorf("the random specs never chose %s", strings.Join(slices.Sorted(slices.Values(missing)), ", "))
	}
}

func appendIfUnseen(missing []string, seen map[string]bool, key string) []string {
	if !seen[key] {
		return append(missing, key)
	}
	return missing
}

func TestTheWellFormedCheckFindsBrokenFiles(t *testing.T) {
	t.Parallel()

	file := func(content string) blueprint.File { return blueprint.File{Content: []byte(content), Mode: FileMode} }
	bp := blueprint.Blueprint{Files: map[string]blueprint.File{
		"shop/__init__.py":          file(""),
		"shop/settings/__init__.py": file(""),
		"shop/settings/base.py":     file("import os\n"),
		"shop/settings/local.py":    file("from .base import *\nfrom ..urls import urlpatterns\n"),
		"shop/views.py":             file("from shop.forms import BookForm\nfrom django.http import HttpResponse\n"),
		"package.json":              file("{\"name\": }\n"),
		"README.md":                 file("Run [[.Name]] now\n"),
		"start-dev.sh":              file("#!/bin/sh"),
	}}
	got := strings.Join(wellFormed(bp), "\n")
	for _, want := range []string{
		"shop/settings/local.py imports ..urls",
		"shop/views.py imports shop.forms",
		"package.json does not parse",
		"README.md carries template residue",
		"start-dev.sh does not end with a newline",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the check missed %q; it reported:\n%s", want, got)
		}
	}
	if strings.Contains(got, ".base") || strings.Contains(got, "django.http") {
		t.Errorf("the check flagged an import that resolves or is not first-party:\n%s", got)
	}
}

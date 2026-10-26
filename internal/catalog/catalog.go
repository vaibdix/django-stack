package catalog

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pelletier/go-toml/v2"
)

//go:embed catalog.toml
var catalogTOML []byte

var loaded = sync.OnceValue(func() Catalog {
	c, err := Parse(catalogTOML)
	if err != nil {
		panic("catalog: " + err.Error())
	}
	return c
})

type Catalog struct {
	Django           []DjangoSeries    `toml:"django"`
	Stacks           []Stack           `toml:"stack"`
	Databases        []Database        `toml:"database"`
	AddOns           []AddOn           `toml:"addon"`
	Tools            Tools             `toml:"tools"`
	Host             []HostRule        `toml:"host"`
	InstalledModules map[string]string `toml:"installed_modules"`
	ScaffoldNames    map[string]string `toml:"scaffold_names"`
}

type DjangoSeries struct {
	Series           string `toml:"series"`
	Python           [2]int `toml:"python"`
	LTS              bool   `toml:"lts"`
	SupportedThrough string `toml:"supported_through"`
}

type Stack struct {
	Name      string `toml:"name"`
	Label     string `toml:"label"`
	Templates bool   `toml:"templates"`
	Tailwind  bool   `toml:"tailwind"`
	HTMX      bool   `toml:"htmx"`
}

type Database struct {
	Name         string   `toml:"name"`
	Label        string   `toml:"label"`
	Option       string   `toml:"option"`
	Image        string   `toml:"image"`
	Driver       string   `toml:"driver"`
	Engine       string   `toml:"engine"`
	Host         string   `toml:"host"`
	Port         int      `toml:"port"`
	Options      []string `toml:"options"`
	UpsertTarget bool     `toml:"upsert_target"`
}

type AddOn struct {
	Name      string   `toml:"name"`
	Label     string   `toml:"label"`
	Requires  []string `toml:"requires"`
	Conflicts []string `toml:"conflicts"`
}

type Tools struct {
	Tailwind        string `toml:"tailwind"`
	DjLint          string `toml:"djlint"`
	PreCommitHooks  string `toml:"pre_commit_hooks"`
	DjangoUpgrade   string `toml:"django_upgrade"`
	DockerEngineAPI string `toml:"docker_engine_api"`
}

type HostRule struct {
	When  string   `toml:"when"`
	Needs []string `toml:"needs"`
}

func (r HostRule) Condition() (kind, name string) {
	kind, name, _ = strings.Cut(r.When, ":")
	return kind, name
}

type Need struct {
	Text    string
	Tools   []string
	Minimum int
}

func ParseNeed(text string) (Need, error) {
	tool, minimum, versioned := strings.Cut(text, ">=")
	need := Need{Text: text, Tools: strings.Split(tool, "|")}
	if slices.Contains(need.Tools, "") {
		return Need{}, fmt.Errorf("host need %q names an empty tool", text)
	}
	if versioned {
		major, err := strconv.Atoi(minimum)
		if err != nil || len(need.Tools) != 1 {
			return Need{}, fmt.Errorf("host need %q must be one tool and a major version, like node>=20", text)
		}
		need.Minimum = major
	}
	return need, nil
}

func (c Catalog) MinimumMajor(tool string) int {
	for _, rule := range c.Host {
		for _, text := range rule.Needs {
			if need, err := ParseNeed(text); err == nil && need.Minimum > 0 && need.Tools[0] == tool {
				return need.Minimum
			}
		}
	}
	return 0
}

func Load() Catalog { return loaded() }

func Parse(data []byte) (Catalog, error) {
	var c Catalog
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		if strict, ok := errors.AsType[*toml.StrictMissingError](err); ok {
			return Catalog{}, errors.New(strict.String())
		}
		return Catalog{}, err
	}
	return c, c.validate()
}

func (d DjangoSeries) SecurityUntil() time.Time {
	month, err := time.Parse("2006-01", d.SupportedThrough)
	if err != nil {
		return time.Time{}
	}
	return month.AddDate(0, 1, 0)
}

func (c Catalog) validate() error {
	var problems []error
	for _, series := range c.Django {
		if series.SecurityUntil().IsZero() {
			problems = append(problems, fmt.Errorf("django %s: supported_through %q is not a year and month like 2027-12", series.Series, series.SupportedThrough))
		}
		if series.Python[0] > series.Python[1] {
			problems = append(problems, fmt.Errorf("django %s: python range %v is reversed", series.Series, series.Python))
		}
	}
	problems = append(problems, unique("django", c.Django, func(d DjangoSeries) string { return d.Series })...)
	problems = append(problems, unique("stack", c.Stacks, func(s Stack) string { return s.Name })...)
	problems = append(problems, unique("database", c.Databases, func(d Database) string { return d.Name })...)
	problems = append(problems, unique("addon", c.AddOns, func(a AddOn) string { return a.Name })...)
	for _, addOn := range c.AddOns {
		for _, name := range slices.Concat(addOn.Requires, addOn.Conflicts) {
			if c.addOn(name) == nil {
				problems = append(problems, fmt.Errorf("addon %s names %q, which is not an add-on", addOn.Name, name))
			}
		}
		for _, name := range addOn.Conflicts {
			if other := c.addOn(name); other != nil && !slices.Contains(other.Conflicts, addOn.Name) {
				problems = append(problems, fmt.Errorf("addon %s conflicts with %s, but %s does not list %s", addOn.Name, name, name, addOn.Name))
			}
		}
	}
	problems = append(problems, c.validateTools()...)
	for _, rule := range c.Host {
		problems = append(problems, c.validateHost(rule)...)
	}
	return errors.Join(problems...)
}

func (c Catalog) validateTools() []error {
	var problems []error
	for name, value := range map[string]string{
		"tailwind": c.Tools.Tailwind, "djlint": c.Tools.DjLint, "pre_commit_hooks": c.Tools.PreCommitHooks,
		"django_upgrade": c.Tools.DjangoUpgrade, "docker_engine_api": c.Tools.DockerEngineAPI,
	} {
		if value == "" {
			problems = append(problems, fmt.Errorf("tools.%s is not set", name))
		}
	}
	return problems
}

func (c Catalog) validateHost(rule HostRule) []error {
	var problems []error
	kind, name := rule.Condition()
	switch kind {
	case "addon":
		if c.addOn(name) == nil {
			problems = append(problems, fmt.Errorf("host rule %q names an unknown add-on", rule.When))
		}
	case "stack":
		if !slices.ContainsFunc(c.Stacks, func(s Stack) bool { return s.trait(name) }) {
			problems = append(problems, fmt.Errorf("host rule %q names a trait no stack has", rule.When))
		}
	case "database":
		if name != "docker" {
			problems = append(problems, fmt.Errorf("host rule %q: a database condition must be database:docker", rule.When))
		}
	default:
		problems = append(problems, fmt.Errorf("host rule %q must name an addon, stack or database condition", rule.When))
	}
	if len(rule.Needs) == 0 {
		problems = append(problems, fmt.Errorf("host rule %q needs nothing", rule.When))
	}
	for _, text := range rule.Needs {
		if _, err := ParseNeed(text); err != nil {
			problems = append(problems, err)
		}
	}
	return problems
}

func (s Stack) trait(name string) bool {
	switch name {
	case "templates":
		return s.Templates
	case "tailwind":
		return s.Tailwind
	case "htmx":
		return s.HTMX
	default:
		return false
	}
}

func (s Stack) Has(trait string) bool { return s.trait(trait) }

func (c Catalog) addOn(name string) *AddOn {
	i := slices.IndexFunc(c.AddOns, func(a AddOn) bool { return a.Name == name })
	if i < 0 {
		return nil
	}
	return &c.AddOns[i]
}

func unique[T any](table string, rows []T, key func(T) string) []error {
	var problems []error
	seen := map[string]bool{}
	for _, row := range rows {
		name := key(row)
		if name == "" || seen[name] {
			problems = append(problems, fmt.Errorf("%s %q is empty or listed twice", table, name))
		}
		seen[name] = true
	}
	return problems
}

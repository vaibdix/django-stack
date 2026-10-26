package catalog

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"slices"
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
	return errors.Join(problems...)
}

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

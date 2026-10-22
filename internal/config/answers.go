package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/tailscale/hujson"

	"github.com/vaibdix/django-cli/internal/version"
)

var AddOnNames = map[Feature]string{
	FeatureGit:            "git",
	FeatureRestFramework:  "rest-framework",
	FeatureNinja:          "django-ninja",
	FeatureAllAuth:        "allauth",
	FeatureRuff:           "ruff",
	FeatureMypy:           "mypy",
	FeaturePreCommit:      "pre-commit",
	FeatureMultiEnv:       "multi-env",
	FeaturePydanticConfig: "pydantic",
	FeatureStructlog:      "structlog",
	FeatureSentry:         "sentry",
}

// ValidateFeatures rejects selections the generator cannot build, whichever
// front end made them.
func ValidateFeatures(selected []Feature) error {
	if slices.Contains(selected, FeatureRestFramework) && slices.Contains(selected, FeatureNinja) {
		return errors.New("pick one API framework: REST Framework or Django Ninja, not both")
	}
	return nil
}

// IsStableDjangoVersion reports whether v is a final release such as 6.1 or
// 6.1.2, not an alpha, beta or release candidate.
func IsStableDjangoVersion(v string) bool {
	parts, err := version.Parse(v)
	return err == nil && (len(parts) == 2 || len(parts) == 3)
}

// DjangoRunsOn reports whether a supported Django series covers this Python
// on the given day. A series past its end of security support is not offered.
func DjangoRunsOn(djangoVersion string, pythonMajor, pythonMinor int, today time.Time) bool {
	if pythonMajor != 3 {
		return false
	}
	series := DjangoSeries(djangoVersion)
	for _, rule := range SupportedDjangoSeries(today) {
		if rule.Series == series {
			return pythonMinor >= rule.MinMinor && pythonMinor <= rule.MaxMinor
		}
	}
	return false
}

// Answers is what a non-interactive run reads instead of asking: the same
// questions the interactive setup asks, and nothing the engine works out for
// itself (credentials, ports, the interpreter's version).
type Answers struct {
	Name     string   `json:"name"`
	Python   string   `json:"python"`
	Django   string   `json:"django"`
	Database Database `json:"database"`
	Stack    Stack    `json:"stack"`
	App      string   `json:"app,omitempty"`
	AddOns   []string `json:"addons,omitempty"`
}

func ReadAnswers(path string) (Answers, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Answers{}, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return DecodeAnswers(data)
}

func DecodeAnswers(data []byte) (Answers, error) {
	standard, err := hujson.Standardize(slices.Clone(data))
	if err != nil {
		return Answers{}, fmt.Errorf("invalid answers file: %w", err)
	}
	var manifest struct {
		Answers json.RawMessage `json:"answers"`
	}
	if json.Unmarshal(standard, &manifest) == nil && len(manifest.Answers) > 0 {
		standard = manifest.Answers
	}

	decoder := json.NewDecoder(bytes.NewReader(standard))
	decoder.DisallowUnknownFields()

	var answers Answers
	if err := decoder.Decode(&answers); err != nil {
		return Answers{}, fmt.Errorf("invalid answers file: %w", err)
	}
	return answers, nil
}

// Project turns the answers into a project configuration, applying the same
// rules the interactive setup enforces. The interpreter is left to the caller,
// which has to run it to learn its version.
func (a Answers) Project() (Project, error) {
	p := Project{
		ProjectName:   a.Name,
		DjangoVersion: a.Django,
		DatabaseType:  a.Database,
		AppName:       a.App,
	}

	if a.Python == "" {
		return Project{}, errors.New(`"python" is required: a Python 3.10+ version such as "3.13", or the path or name of an interpreter`)
	}
	if !IsStableDjangoVersion(a.Django) {
		return Project{}, fmt.Errorf(`"django" must be an exact stable release such as "6.1.2", got %q`, a.Django)
	}
	switch a.Database {
	case SQLite, PostgresDocker, MySQLDocker:
	default:
		return Project{}, fmt.Errorf(`"database" must be one of %s, %s or %s, got %q`, SQLite, PostgresDocker, MySQLDocker, a.Database)
	}
	if !slices.Contains(AllStacks(), a.Stack) {
		return Project{}, fmt.Errorf(`"stack" must be one of %s, got %q`, joinStacks(), a.Stack)
	}

	features, err := featuresFromNames(a.AddOns)
	if err != nil {
		return Project{}, err
	}
	selected := append(a.Stack.Features(), features...)
	if err := ValidateFeatures(selected); err != nil {
		return Project{}, err
	}
	p.ApplyFeatures(selected)

	if err := p.ValidateProjectName(); err != nil {
		return Project{}, fmt.Errorf(`invalid "name": %w`, err)
	}
	if p.NeedsAppName() && p.AppName == "" {
		return Project{}, errors.New(`"app" is required for this stack and these add-ons`)
	}
	if err := p.ValidateAppName(); err != nil {
		return Project{}, fmt.Errorf(`invalid "app": %w`, err)
	}
	return p, nil
}

func featuresFromNames(names []string) ([]Feature, error) {
	byName := make(map[string]Feature, len(AddOnNames))
	for feature, name := range AddOnNames {
		byName[name] = feature
	}

	features := make([]Feature, 0, len(names))
	for _, name := range names {
		feature, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("unknown add-on %q; the add-ons are %s", name, joinAddOns())
		}
		features = append(features, feature)
	}
	return features, nil
}

func joinAddOns() string {
	names := make([]string, 0, len(AddOnNames))
	for _, name := range AddOnNames {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func joinStacks() string {
	stacks := AllStacks()
	names := make([]string, len(stacks))
	for i, stack := range stacks {
		names[i] = string(stack)
	}
	return strings.Join(names, ", ")
}

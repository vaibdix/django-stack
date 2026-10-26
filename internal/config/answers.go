package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/tailscale/hujson"

	"github.com/vaibdix/django-stack/internal/version"
)

func IsStableDjangoVersion(v string) bool {
	parts, err := version.Parse(v)
	return err == nil && (len(parts) == 2 || len(parts) == 3)
}

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

type Answers struct {
	Name     string   `json:"name"`
	Python   string   `json:"python"`
	Django   string   `json:"django"`
	Database Database `json:"database"`
	Stack    Stack    `json:"stack"`
	App      string   `json:"app,omitempty"`
	AddOns   []string `json:"addons,omitempty"`
}

func (c Choices) Answers(python string) Answers {
	enabled := c.EnabledAddOns()
	names := make([]string, len(enabled))
	for i, addOn := range enabled {
		names[i] = string(addOn)
	}
	slices.Sort(names)
	return Answers{
		Name:     c.Name,
		Python:   python,
		Django:   c.Django,
		Database: c.Database,
		Stack:    c.Stack,
		App:      c.App,
		AddOns:   names,
	}
}

func ReadAnswers(path string) (Answers, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Answers{}, fmt.Errorf("read %s: %w", path, err)
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

func (a Answers) Choices() (Choices, error) {
	p := Choices{
		Name:     a.Name,
		Django:   a.Django,
		Database: a.Database,
		App:      a.App,
		Stack:    a.Stack,
	}

	if a.Python == "" {
		return Choices{}, errors.New(`"python" is required: a Python 3.10+ version such as "3.13", or the path or name of an interpreter`)
	}
	if !IsStableDjangoVersion(a.Django) {
		return Choices{}, fmt.Errorf(`"django" must be an exact stable release such as "6.1.2", got %q`, a.Django)
	}
	if !a.Database.Valid() {
		names := make([]string, 0, len(databases))
		for _, d := range Databases() {
			names = append(names, string(d))
		}
		last := len(names) - 1
		return Choices{}, fmt.Errorf(`"database" must be one of %s or %s, got %q`, strings.Join(names[:last], ", "), names[last], a.Database)
	}
	if !a.Stack.Valid() {
		return Choices{}, fmt.Errorf(`"stack" must be one of %s, got %q`, joinStacks(), a.Stack)
	}

	selected, err := ParseAddOns(a.AddOns)
	if err != nil {
		return Choices{}, err
	}
	if err := ValidateAddOns(selected); err != nil {
		return Choices{}, err
	}
	p.AddOns = NewSet(selected...)

	if err := p.ValidateProjectName(PythonNames{}); err != nil {
		return Choices{}, fmt.Errorf(`invalid "name": %w`, err)
	}
	if p.NeedsAppName() && p.App == "" {
		return Choices{}, errors.New(`"app" is required for this stack and these add-ons`)
	}
	if err := p.ValidateAppName(PythonNames{}); err != nil {
		return Choices{}, fmt.Errorf(`invalid "app": %w`, err)
	}
	return p, nil
}

func joinStacks() string {
	stacks := AllStacks()
	names := make([]string, len(stacks))
	for i, stack := range stacks {
		names[i] = string(stack)
	}
	return strings.Join(names, ", ")
}

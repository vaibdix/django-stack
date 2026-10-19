package tui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/pypi"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

type DjangoOption struct {
	Version       string
	UploadedAt    time.Time
	Compatibility string
	Recommended   bool
	LTS           bool
	Label         string
}

const djangoReleaseFetchTimeout = 5 * time.Second

var djangoReleaseHTTPClient = &http.Client{Timeout: djangoReleaseFetchTimeout}

// fetchDjangoReleaseOptions is a seam for tests. today decides which series
// are still supported, so a test pins it instead of reading the wall clock.
var fetchDjangoReleaseOptions = fetchDjangoReleaseOptionsFromPyPI

type djangoOptionsMsg struct {
	Options []DjangoOption
	Note    string
}

func (m *Model) fetchDjangoOptionsCmd() tea.Cmd {
	pythonOption, ok := m.selectedPythonOption()
	if !ok {
		return func() tea.Msg {
			return djangoOptionsMsg{Note: "Choose a Python interpreter first."}
		}
	}
	today := m.Clock()
	run := m.runner()
	return func() tea.Msg {
		// The releases offered come from the index the install will use.
		uv := pyenv.DetectToolchain(context.Background(), run) == pyenv.UV
		options, note := fetchDjangoReleaseOptions(pythonOption, today, pypi.IndexFor(uv, os.Getenv))
		return djangoOptionsMsg{Options: options, Note: note}
	}
}

func (m *Model) applyDjangoOptions(options []DjangoOption, note string) {
	m.AvailableDjangoOptions = options
	m.DjangoVersionNote = note

	if len(options) == 0 {
		m.DjangoVersion = ""
		m.DjangoVersionForm = buildUnavailableDjangoForm(&m.DjangoVersion, note)
		return
	}

	if m.DjangoVersion == "" || !m.hasDjangoOption(m.DjangoVersion) {
		m.DjangoVersion = options[0].Version
	}
	m.DjangoVersionForm = m.buildDjangoVersionForm()
}

func buildPendingDjangoForm(target *string) *huh.Form {
	return newForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Django Version").
				Description("Fetching the latest compatible Django releases...").
				Options(huh.NewOption("Loading...", "")).
				Validate(requireDjango).
				Value(target),
		),
	)
}

func (m *Model) buildDjangoVersionForm() *huh.Form {
	options := make([]huh.Option[string], 0, len(m.AvailableDjangoOptions))
	for _, option := range m.AvailableDjangoOptions {
		options = append(options, huh.NewOption(option.Label, option.Version))
	}

	description := "Pick one of the latest compatible stable Django releases."
	if m.DjangoVersionNote != "" {
		description = m.DjangoVersionNote
	}

	return newForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Django Version").
				Description(description).
				Options(options...).
				Validate(requireDjango).
				Value(&m.DjangoVersion),
		),
	)
}

func buildUnavailableDjangoForm(target *string, description string) *huh.Form {
	if description == "" {
		description = "No compatible Django versions are available for the selected Python interpreter."
	}
	if target != nil {
		*target = ""
	}
	return newForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Django Version").
				Description(description).
				Options(huh.NewOption("No compatible stable Django release found", "")).
				Validate(requireDjango).
				Value(target),
		),
	)
}

func (m *Model) hasDjangoOption(version string) bool {
	for _, option := range m.AvailableDjangoOptions {
		if option.Version == version {
			return true
		}
	}
	return false
}

func fetchDjangoReleaseOptionsFromPyPI(pythonOption pyenv.Option, today time.Time, index string) ([]DjangoOption, string) {
	if note, unsupported := unsupportedPythonNote(pythonOption, today); unsupported {
		return nil, note
	}

	options, err := fetchLiveDjangoReleaseOptions(pythonOption, today, index)
	if err != nil {
		// Without PyPI the install step cannot run either, so a list of
		// releases bundled at build time would only defer the failure.
		return nil, fmt.Sprintf("The Django releases could not be read from %s (%v). "+
			"Check the connection, then press ← and choose the interpreter again to retry.", index, err)
	}
	if len(options) == 0 {
		return nil, fmt.Sprintf("PyPI lists no supported Django release for Python %d.%d.", pythonOption.Major, pythonOption.Minor)
	}
	return options, fmt.Sprintf("The newest release of each Django series that supports Python %d.%d, from PyPI.",
		pythonOption.Major, pythonOption.Minor)
}

func fetchLiveDjangoReleaseOptions(pythonOption pyenv.Option, today time.Time, index string) ([]DjangoOption, error) {
	ctx, cancel := context.WithTimeout(context.Background(), djangoReleaseFetchTimeout)
	defer cancel()

	releases, err := pypi.Fetch(ctx, djangoReleaseHTTPClient, index, "django")
	if err != nil {
		return nil, err
	}
	return buildDjangoOptionsFromReleases(pythonOption, releases, today), nil
}

func buildDjangoOptionsFromReleases(pythonOption pyenv.Option, releases []pypi.Release, today time.Time) []DjangoOption {
	versions := make([]string, 0, len(releases))
	uploads := make(map[string]time.Time, len(releases))
	for _, release := range releases {
		if !pypi.SupportsPython(release.RequiresPython, pythonOption.Major, pythonOption.Minor) {
			continue
		}
		versions = append(versions, release.Version)
		uploads[release.Version] = release.UploadedAt
	}

	options := newestReleasePerSeries(pythonOption, versions, today)
	for i := range options {
		options[i].UploadedAt = uploads[options[i].Version]
	}
	return options
}

func newestReleasePerSeries(pythonOption pyenv.Option, versions []string, today time.Time) []DjangoOption {
	newest := map[string]string{}
	for _, version := range versions {
		if !config.IsStableDjangoVersion(version) ||
			!config.DjangoRunsOn(version, pythonOption.Major, pythonOption.Minor, today) {
			continue
		}
		series := djangoSeries(version)
		if current, seen := newest[series]; !seen || compareVersionStrings(version, current) > 0 {
			newest[series] = version
		}
	}

	options := make([]DjangoOption, 0, len(newest))
	for _, rule := range config.SupportedDjangoSeries(today) {
		version, ok := newest[rule.Series]
		if !ok {
			continue
		}
		options = append(options, DjangoOption{
			Version:       version,
			LTS:           rule.LTS,
			Compatibility: compatibilityLabel(pythonOption),
		})
	}

	sort.SliceStable(options, func(i, j int) bool {
		return compareVersionStrings(options[i].Version, options[j].Version) > 0
	})
	for i := range options {
		options[i].Recommended = i == 0
		options[i].Label = buildDjangoOptionLabel(options[i])
	}
	return options
}
func buildDjangoOptionLabel(option DjangoOption) string {
	label := option.Version
	if option.LTS {
		label += " LTS"
	}
	if option.Recommended {
		label += " [recommended]"
	}
	if option.Compatibility != "" {
		label += " - " + option.Compatibility
	}
	return label
}
func compatibilityLabel(pythonOption pyenv.Option) string {
	return fmt.Sprintf("compatible with Python %d.%d", pythonOption.Major, pythonOption.Minor)
}

func djangoSeries(version string) string {
	return config.DjangoSeries(version)
}

func compareVersionStrings(left, right string) int {
	leftParts := parseVersionParts(left)
	rightParts := parseVersionParts(right)
	for i := 0; i < len(leftParts) || i < len(rightParts); i++ {
		leftValue := 0
		rightValue := 0
		if i < len(leftParts) {
			leftValue = leftParts[i]
		}
		if i < len(rightParts) {
			rightValue = rightParts[i]
		}
		if leftValue > rightValue {
			return 1
		}
		if leftValue < rightValue {
			return -1
		}
	}
	return 0
}

func parseVersionParts(version string) []int {
	parts := strings.Split(version, ".")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		value := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				break
			}
			value = value*10 + int(r-'0')
		}
		values = append(values, value)
	}
	return values
}

func requireDjango(version string) error {
	if version == "" {
		return errors.New("no Django release to install for this interpreter; press ← and pick another one")
	}
	return nil
}

func unsupportedPythonNote(pythonOption pyenv.Option, today time.Time) (string, bool) {
	newest := config.NewestSupportedPythonMinor(today)
	if pythonOption.Major == 3 && pythonOption.Minor > newest {
		return fmt.Sprintf("No supported Django release runs on Python %d.%d yet; the newest it supports is 3.%d. "+
			"Press ← and pick an older interpreter.", pythonOption.Major, pythonOption.Minor, newest), true
	}
	return "", false
}

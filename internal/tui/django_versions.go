package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/config"
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

type djangoReleaseIndex struct {
	Releases map[string][]djangoReleaseFile `json:"releases"`
}

type djangoReleaseFile struct {
	UploadTimeISO8601 string `json:"upload_time_iso_8601"`
	Yanked            bool   `json:"yanked"`
}

var stableDjangoVersionPattern = regexp.MustCompile(`^\d+\.\d+(?:\.\d+)?$`)

var now = time.Now

var fallbackDjangoReleases = []string{
	"6.1",
	"6.0.3",
	"5.2.12",
	"6.0.2",
	"5.2.11",
	"6.0.1",
	"5.2.10",
	"6.0",
	"5.2.9",
	"5.2.8",
}

const djangoReleaseFetchTimeout = 3 * time.Second

var djangoReleaseHTTPClient = &http.Client{Timeout: djangoReleaseFetchTimeout}
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
	return func() tea.Msg {
		options, note := fetchDjangoReleaseOptions(pythonOption)
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

func fetchDjangoReleaseOptionsFromPyPI(pythonOption pyenv.Option) ([]DjangoOption, string) {
	if note, unsupported := unsupportedPythonNote(pythonOption); unsupported {
		return nil, note
	}

	options, err := fetchLiveDjangoReleaseOptions(pythonOption)
	if err == nil && len(options) > 0 {
		return options, fmt.Sprintf("The newest release of each Django series that supports Python %d.%d, from PyPI.",
			pythonOption.Major, pythonOption.Minor)
	}

	return fallbackDjangoOptionsForPython(pythonOption),
		"PyPI could not be reached, so these are the releases bundled with django-stack; a newer patch may exist."
}

func fetchLiveDjangoReleaseOptions(pythonOption pyenv.Option) ([]DjangoOption, error) {
	ctx, cancel := context.WithTimeout(context.Background(), djangoReleaseFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://pypi.org/pypi/Django/json", nil)
	if err != nil {
		return nil, err
	}

	resp, err := djangoReleaseHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected PyPI status: %s", resp.Status)
	}

	var index djangoReleaseIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, err
	}

	return buildDjangoOptionsFromReleaseIndex(pythonOption, index.Releases), nil
}

func buildDjangoOptionsFromReleaseIndex(pythonOption pyenv.Option, releases map[string][]djangoReleaseFile) []DjangoOption {
	versions := make([]string, 0, len(releases))
	uploads := make(map[string]time.Time, len(releases))
	for version, files := range releases {
		uploadedAt, ok := latestNonYankedUploadTime(files)
		if !ok {
			continue
		}
		versions = append(versions, version)
		uploads[version] = uploadedAt
	}

	options := newestReleasePerSeries(pythonOption, versions)
	for i := range options {
		options[i].UploadedAt = uploads[options[i].Version]
	}
	return options
}
func latestNonYankedUploadTime(files []djangoReleaseFile) (time.Time, bool) {
	var latest time.Time
	found := false
	for _, file := range files {
		if file.Yanked || file.UploadTimeISO8601 == "" {
			continue
		}
		uploadedAt, err := time.Parse(time.RFC3339, file.UploadTimeISO8601)
		if err != nil {
			continue
		}
		if !found || uploadedAt.After(latest) {
			latest = uploadedAt
			found = true
		}
	}
	return latest, found
}

func fallbackDjangoOptionsForPython(pythonOption pyenv.Option) []DjangoOption {
	return newestReleasePerSeries(pythonOption, fallbackDjangoReleases)
}

func newestReleasePerSeries(pythonOption pyenv.Option, versions []string) []DjangoOption {
	newest := map[string]string{}
	for _, version := range versions {
		if !isStableDjangoVersion(version) || !isCompatibleDjangoVersion(pythonOption, version) {
			continue
		}
		series := djangoSeries(version)
		if current, seen := newest[series]; !seen || compareVersionStrings(version, current) > 0 {
			newest[series] = version
		}
	}

	options := make([]DjangoOption, 0, len(newest))
	for _, rule := range config.SupportedDjangoSeries(now()) {
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

func isStableDjangoVersion(version string) bool {
	return stableDjangoVersionPattern.MatchString(version)
}

func isCompatibleDjangoVersion(pythonOption pyenv.Option, version string) bool {
	series := djangoSeries(version)
	if pythonOption.Major != 3 {
		return false
	}
	for _, rule := range config.SupportedDjangoSeries(now()) {
		if rule.Series == series {
			return pythonOption.Minor >= rule.MinMinor && pythonOption.Minor <= rule.MaxMinor
		}
	}
	return false
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

func unsupportedPythonNote(pythonOption pyenv.Option) (string, bool) {
	newest := config.NewestSupportedPythonMinor(now())
	if pythonOption.Major == 3 && pythonOption.Minor > newest {
		return fmt.Sprintf("No supported Django release runs on Python %d.%d yet; the newest it supports is 3.%d. "+
			"Press ← and pick an older interpreter.", pythonOption.Major, pythonOption.Minor, newest), true
	}
	return "", false
}

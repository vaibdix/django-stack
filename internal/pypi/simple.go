package pypi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	SimpleIndex = "https://pypi.org/simple/"

	simpleJSON = "application/vnd.pypi.simple.v1+json"
)

type Release struct {
	Version        string
	UploadedAt     time.Time
	RequiresPython string
}

type simpleProject struct {
	Versions []string     `json:"versions"`
	Files    []simpleFile `json:"files"`
}

type simpleFile struct {
	Filename       string          `json:"filename"`
	RequiresPython string          `json:"requires-python"`
	UploadTime     string          `json:"upload-time"`
	Yanked         json.RawMessage `json:"yanked"`
}

func (f simpleFile) yanked() bool {
	text := strings.TrimSpace(string(f.Yanked))
	return text != "" && text != "false" && text != "null"
}

func Fetch(ctx context.Context, client *http.Client, project string) ([]Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, SimpleIndex+project+"/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", simpleJSON)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected PyPI status: %s", resp.Status)
	}

	var index simpleProject
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, err
	}
	return releasesFrom(project, index), nil
}

func releasesFrom(project string, index simpleProject) []Release {
	byVersion := make(map[string]*Release, len(index.Versions))
	for _, file := range index.Files {
		if file.yanked() {
			continue
		}
		version, ok := versionFromFilename(project, file.Filename)
		if !ok {
			continue
		}
		uploadedAt, _ := time.Parse(time.RFC3339Nano, file.UploadTime)

		release, seen := byVersion[version]
		if !seen {
			release = &Release{Version: version}
			byVersion[version] = release
		}
		if uploadedAt.After(release.UploadedAt) {
			release.UploadedAt = uploadedAt
		}
		if release.RequiresPython == "" {
			release.RequiresPython = file.RequiresPython
		}
	}

	releases := make([]Release, 0, len(byVersion))
	for _, version := range index.Versions {
		if release, ok := byVersion[version]; ok {
			releases = append(releases, *release)
		}
	}
	return releases
}

func versionFromFilename(project, filename string) (string, bool) {
	normalized := strings.ToLower(strings.ReplaceAll(project, "-", "_")) + "-"
	lower := strings.ToLower(filename)
	if !strings.HasPrefix(lower, normalized) {
		return "", false
	}
	rest := filename[len(normalized):]

	switch {
	case strings.HasSuffix(rest, ".whl"):
		version, _, ok := strings.Cut(rest, "-")
		return version, ok
	case strings.HasSuffix(rest, ".tar.gz"):
		return strings.TrimSuffix(rest, ".tar.gz"), true
	case strings.HasSuffix(rest, ".zip"):
		return strings.TrimSuffix(rest, ".zip"), true
	default:
		return "", false
	}
}

func SupportsPython(requiresPython string, major, minor int) bool {
	for spec := range strings.SplitSeq(requiresPython, ",") {
		spec = strings.TrimSpace(spec)
		bound, ok := strings.CutPrefix(spec, ">=")
		if !ok {
			continue
		}
		boundMajor, boundMinor, ok := majorMinor(bound)
		if !ok {
			continue
		}
		if major < boundMajor || (major == boundMajor && minor < boundMinor) {
			return false
		}
	}
	return true
}

func majorMinor(version string) (int, int, bool) {
	parts := strings.SplitN(strings.TrimSpace(version), ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minor, err := strconv.Atoi(strings.TrimRight(parts[1], "*"))
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}

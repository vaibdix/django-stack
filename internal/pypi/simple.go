package pypi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultIndex is where pip and uv install from when nothing else is
	// configured.
	DefaultIndex = "https://pypi.org/simple/"

	simpleJSON = "application/vnd.pypi.simple.v1+json"
)

// IndexFor is the Simple API base the installer will download from, so the
// releases offered and the hashes pinned describe the files it installs. uv
// reads UV_DEFAULT_INDEX (UV_INDEX_URL before it) and never PIP_INDEX_URL;
// pip reads PIP_INDEX_URL. Indexes set only in a pip.conf or uv.toml are not
// read here; set the variable to use one.
func IndexFor(uv bool, getenv func(string) string) string {
	variables := []string{"PIP_INDEX_URL"}
	if uv {
		variables = []string{"UV_DEFAULT_INDEX", "UV_INDEX_URL"}
	}
	for _, name := range variables {
		if index := strings.TrimSpace(getenv(name)); index != "" {
			return strings.TrimRight(index, "/") + "/"
		}
	}
	return DefaultIndex
}

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
	Filename       string            `json:"filename"`
	Hashes         map[string]string `json:"hashes"`
	RequiresPython string            `json:"requires-python"`
	UploadTime     string            `json:"upload-time"`
	Yanked         json.RawMessage   `json:"yanked"`
}

func (f simpleFile) yanked() bool {
	text := strings.TrimSpace(string(f.Yanked))
	return text != "" && text != "false" && text != "null"
}

func Fetch(ctx context.Context, client *http.Client, indexURL, project string) ([]Release, error) {
	index, err := fetchIndex(ctx, client, indexURL, project)
	if err != nil {
		return nil, err
	}
	return releasesFrom(project, index), nil
}

// FileHashes returns the sha256 of every file PyPI publishes for one version
// of a project, sorted, in the form pip's --hash option takes. pip checks the
// one file it picks against this list, so every wheel and the sdist belong in it.
func FileHashes(ctx context.Context, client *http.Client, indexURL, project, version string) ([]string, error) {
	index, err := fetchIndex(ctx, client, indexURL, project)
	if err != nil {
		return nil, err
	}
	var hashes []string
	for _, file := range index.Files {
		name, fileVersion, ok := splitDistributionFilename(file.Filename)
		if !ok || normalizeName(name) != normalizeName(project) || fileVersion != version {
			continue
		}
		if sha, ok := file.Hashes["sha256"]; ok {
			hashes = append(hashes, "sha256:"+sha)
		}
	}
	if len(hashes) == 0 {
		return nil, fmt.Errorf("PyPI lists no sha256 for %s==%s", project, version)
	}
	sort.Strings(hashes)
	return hashes, nil
}

func splitDistributionFilename(filename string) (name, version string, ok bool) {
	if base, isWheel := strings.CutSuffix(filename, ".whl"); isWheel {
		parts := strings.Split(base, "-")
		if len(parts) < 5 {
			return "", "", false
		}
		return parts[0], parts[1], true
	}
	for _, ext := range []string{".tar.gz", ".zip"} {
		if base, isSdist := strings.CutSuffix(filename, ext); isSdist {
			at := strings.LastIndex(base, "-")
			if at <= 0 {
				return "", "", false
			}
			return base[:at], base[at+1:], true
		}
	}
	return "", "", false
}

// normalizeName applies PEP 503: runs of "-", "_" and "." are equivalent.
func normalizeName(name string) string {
	var b strings.Builder
	separator := false
	for _, r := range strings.ToLower(name) {
		if r == '-' || r == '_' || r == '.' {
			separator = true
			continue
		}
		if separator && b.Len() > 0 {
			b.WriteByte('-')
		}
		separator = false
		b.WriteRune(r)
	}
	return b.String()
}

func fetchIndex(ctx context.Context, client *http.Client, indexURL, project string) (simpleProject, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL+project+"/", nil)
	if err != nil {
		return simpleProject{}, err
	}
	req.Header.Set("Accept", simpleJSON)

	resp, err := client.Do(req)
	if err != nil {
		return simpleProject{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return simpleProject{}, fmt.Errorf("%s answered %s for %s", indexURL, resp.Status, project)
	}
	// An index that only serves the HTML form of the Simple API (PEP 503)
	// cannot be read for releases or hashes, and guessing would be worse.
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, simpleJSON) {
		return simpleProject{}, fmt.Errorf("%s does not serve the JSON Simple API (PEP 691): it answered with %q", indexURL, contentType)
	}

	var index simpleProject
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return simpleProject{}, err
	}
	return index, nil
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

package pypi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/version"
)

const (
	DefaultIndex = "https://pypi.org/simple/"

	simpleJSON = "application/vnd.pypi.simple.v1+json"
)

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

const (
	fetchAttempts = 4
	firstBackoff  = 500 * time.Millisecond
)

func fetchIndex(ctx context.Context, client *http.Client, indexURL, project string) (simpleProject, error) {
	var err error
	for attempt := range fetchAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return simpleProject{}, errors.Join(err, ctx.Err())
			case <-time.After(firstBackoff << (attempt - 1)):
			}
		}
		var index simpleProject
		var transient bool
		index, transient, err = fetchIndexOnce(ctx, client, indexURL, project)
		if err == nil || !transient {
			return index, err
		}
	}
	return simpleProject{}, err
}

func fetchIndexOnce(ctx context.Context, client *http.Client, indexURL, project string) (simpleProject, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL+project+"/", nil)
	if err != nil {
		return simpleProject{}, false, err
	}
	req.Header.Set("Accept", simpleJSON)

	resp, err := client.Do(req)
	if err != nil {
		return simpleProject{}, ctx.Err() == nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		transient := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
		return simpleProject{}, transient, fmt.Errorf("%s answered %s for %s", indexURL, resp.Status, project)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, simpleJSON) {
		return simpleProject{}, false, fmt.Errorf("%s does not serve the JSON Simple API (PEP 691): it answered with %q", indexURL, contentType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return simpleProject{}, ctx.Err() == nil, err
	}
	var index simpleProject
	if err := json.Unmarshal(body, &index); err != nil {
		return simpleProject{}, false, err
	}
	return index, false, nil
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

func SupportsPython(requiresPython string, python ...int) (bool, error) {
	for clause := range strings.SplitSeq(requiresPython, ",") {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}
		ok, err := satisfies(python, clause)
		if err != nil {
			return false, fmt.Errorf("requires-python %q: %w", requiresPython, err)
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

var specifierOperators = []string{"===", "~=", "==", "!=", "<=", ">=", "<", ">"}

func satisfies(python []int, clause string) (bool, error) {
	for _, operator := range specifierOperators {
		bound, found := strings.CutPrefix(clause, operator)
		if !found {
			continue
		}
		bound = strings.TrimSpace(bound)
		prefix, wildcard := strings.CutSuffix(bound, ".*")
		parts, err := version.Parse(prefix)
		if err != nil {
			return false, fmt.Errorf("clause %q: %w", clause, err)
		}

		switch {
		case wildcard && (operator == "==" || operator == "!="):
			match := version.HasPrefix(python, parts)
			return match == (operator == "=="), nil
		case wildcard:
			return false, fmt.Errorf("clause %q: a wildcard only goes with == or !=", clause)
		}

		order := version.Compare(python, parts)
		switch operator {
		case "==", "===":
			return order == 0, nil
		case "!=":
			return order != 0, nil
		case "<=":
			return order <= 0, nil
		case ">=":
			return order >= 0, nil
		case "<":
			return order < 0, nil
		case ">":
			return order > 0, nil
		case "~=":
			if len(parts) < 2 {
				return false, fmt.Errorf("clause %q: ~= needs at least two release components", clause)
			}
			return order >= 0 && version.HasPrefix(python, parts[:len(parts)-1]), nil
		}
	}
	return false, fmt.Errorf("clause %q has no comparison operator", clause)
}

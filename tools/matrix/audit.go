package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type auditReport struct {
	Dependencies []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Vulns   []struct {
			ID          string   `json:"id"`
			FixVersions []string `json:"fix_versions"`
		} `json:"vulns"`
	} `json:"dependencies"`
}

func (c *checker) audit(ctx context.Context) {
	report := filepath.Join(c.logDir, "pip-audit.json")
	audit := []string{"uvx", "--quiet", "pip-audit", "--progress-spinner", "off", "--disable-pip", "--format", "json", "--output", report}
	if c.exists("uv.lock") {
		exported := filepath.Join(c.logDir, "audit-requirements.txt")
		c.cmd(ctx, "pip-audit", nil, "sh", "-c", "uv export --frozen --all-groups --no-hashes --format requirements-txt -o "+
			shellJoin([]string{exported})+" >/dev/null && "+shellJoin(append(audit, "--no-deps", "-r", exported)))
	} else {
		args := append(slices.Clone(audit), "--require-hashes")
		for _, file := range []string{"requirements.txt", "requirements-dev.txt"} {
			if c.exists(file) {
				args = append(args, "-r", file)
			}
		}
		c.cmd(ctx, "pip-audit", nil, args...)
	}
	res := &c.results[len(c.results)-1]
	if res.OK {
		return
	}
	late, err := c.fixesReleasedAfterCutoff(ctx, report)
	if err != nil || len(late) == 0 {
		return
	}
	res.OK = true
	res.Note = "no fix installable on this Python by the run's package date: " + strings.Join(late, ", ")
}

func (c *checker) fixesReleasedAfterCutoff(ctx context.Context, report string) ([]string, error) {
	data, err := os.ReadFile(report)
	if err != nil {
		return nil, err
	}
	var parsed auditReport
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	cutoff, err := time.Parse(time.RFC3339, c.r.matrix.ExcludeNewer)
	if err != nil {
		return nil, err
	}
	pythonVersion, err := os.ReadFile(filepath.Join(c.dir, ".python-version"))
	if err != nil {
		return nil, err
	}
	var late []string
	for _, dep := range parsed.Dependencies {
		for _, vuln := range dep.Vulns {
			available := false
			for _, fix := range vuln.FixVersions {
				release, err := releaseOf(ctx, dep.Name, fix)
				if err != nil {
					return nil, err
				}
				available = available || (!release.uploaded.After(cutoff) && pythonAllowed(release.requiresPython, strings.TrimSpace(string(pythonVersion))))
			}
			if available || len(vuln.FixVersions) == 0 {
				return nil, nil
			}
			late = append(late, fmt.Sprintf("%s %s %s", dep.Name, dep.Version, vuln.ID))
		}
	}
	return late, nil
}

type release struct {
	uploaded       time.Time
	requiresPython string
}

func releaseOf(ctx context.Context, project, version string) (release, error) {
	endpoint := "https://pypi.org/pypi/" + url.PathEscape(project) + "/" + url.PathEscape(version) + "/json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return release{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return release{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return release{}, fmt.Errorf("%s: %s", endpoint, resp.Status)
	}
	var body struct {
		Info struct {
			RequiresPython string `json:"requires_python"`
		} `json:"info"`
		URLs []struct {
			Uploaded time.Time `json:"upload_time_iso_8601"`
		} `json:"urls"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return release{}, err
	}
	if len(body.URLs) == 0 {
		return release{}, fmt.Errorf("%s: no files", endpoint)
	}
	first := body.URLs[0].Uploaded
	for _, file := range body.URLs[1:] {
		if file.Uploaded.Before(first) {
			first = file.Uploaded
		}
	}
	return release{uploaded: first, requiresPython: body.Info.RequiresPython}, nil
}

func pythonAllowed(spec, version string) bool {
	have := versionParts(version)
	for clause := range strings.SplitSeq(spec, ",") {
		clause = strings.TrimSpace(clause)
		op := strings.TrimRight(clause[:min(len(clause), 2)], "0123456789.* ")
		want := versionParts(strings.TrimSpace(strings.TrimPrefix(clause, op)))
		if clause == "" || len(want) == 0 {
			continue
		}
		c := slices.Compare(have[:min(len(have), len(want))], want)
		ok := map[string]bool{">=": c >= 0, ">": c > 0, "<=": c <= 0, "<": c < 0, "==": c == 0, "!=": c != 0}
		if allowed, known := ok[op]; known && !allowed {
			return false
		}
	}
	return true
}

func versionParts(version string) []int {
	var parts []int
	for field := range strings.SplitSeq(version, ".") {
		n, err := strconv.Atoi(field)
		if err != nil {
			break
		}
		parts = append(parts, n)
	}
	return parts
}

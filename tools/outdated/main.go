package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
)

type pin struct {
	name    string
	file    string
	pattern string
	latest  func(ctx context.Context, pinned string) (string, error)
	behind  func(pinned, latest string) bool
}

type finding struct {
	name, pinned, latest, status string
}

func main() {
	root := flag.String("root", ".", "repository root")
	var holds stringList
	flag.Var(&holds, "hold", "a pin that may stay behind, as name (repeatable)")
	flag.Parse()

	if err := run(*root, holds); err != nil {
		fmt.Fprintln(os.Stderr, "outdated:", err)
		os.Exit(1)
	}
}

func run(root string, holds []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	findings, err := check(ctx, root, holds)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tPINNED\tLATEST\tSTATUS")
	failed := false
	for _, f := range findings {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", f.name, f.pinned, f.latest, f.status)
		failed = failed || f.status == "behind"
	}
	_ = w.Flush()
	if failed {
		return errors.New("some pins are behind; bump them or add them to OUTDATED_HOLDS")
	}
	return nil
}

func check(ctx context.Context, root string, holds []string) ([]finding, error) {
	pins, err := pinsIn(root)
	if err != nil {
		return nil, err
	}
	var findings []finding
	for _, p := range pins {
		pinned, err := readPin(root, p)
		if err != nil {
			return nil, err
		}
		latest, err := p.latest(ctx, pinned)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.name, err)
		}
		status := "ok"
		if p.behind(pinned, latest) {
			status = "behind"
			if slices.Contains(holds, p.name) {
				status = "held"
			}
		}
		findings = append(findings, finding{p.name, pinned, latest, status})
	}
	modules, err := goModules(ctx, root, holds)
	if err != nil {
		return nil, err
	}
	return append(findings, modules...), nil
}

func pinsIn(root string) ([]pin, error) {
	pins := []pin{
		{"golangci-lint", "Makefile", `GOLANGCI_LINT_VERSION = (\S+)`, githubRelease("golangci/golangci-lint"), differ},
		{"tailwindcss", "internal/scaffold/tailwind.go", `tailwindVersion = "([^"]+)"`, npmLatest("tailwindcss"), differ},
		{"pre-commit-hooks", "internal/scaffold/precommit.go", `preCommitHooksRev = "([^"]+)"`, githubRelease("pre-commit/pre-commit-hooks"), differ},
		{"django-upgrade", "internal/scaffold/precommit.go", `djangoUpgradeRev\s+= "([^"]+)"`, pypiLatest("django-upgrade"), differ},
		{"djlint", "internal/scaffold/constants.go", `djLintVersion = "([^"]+)"`, pypiLatest("djlint"), differ},
		{"postgres", "internal/config/database.go", `"postgres:([^"]+)"`, dockerTag("postgres", "latest"), differ},
		{"mysql", "internal/config/database.go", `"mysql:([^"]+)"`, dockerTag("mysql", "lts"), differ},
	}
	workflows, err := filepath.Glob(filepath.Join(root, "ci", "workflows", "*.yml"))
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	action := regexp.MustCompile(`uses: ([\w.-]+/[\w.-]+)@(v\d+)`)
	for _, workflow := range workflows {
		content, err := os.ReadFile(workflow)
		if err != nil {
			return nil, err
		}
		for _, m := range action.FindAllStringSubmatch(string(content), -1) {
			if seen[m[1]] {
				continue
			}
			seen[m[1]] = true
			rel, _ := filepath.Rel(root, workflow)
			pins = append(pins, pin{m[1], rel, regexp.QuoteMeta(m[1]) + `@(v\d+)`, githubRelease(m[1]), majorBehind})
		}
	}
	return pins, nil
}

func readPin(root string, p pin) (string, error) {
	content, err := os.ReadFile(filepath.Join(root, p.file))
	if err != nil {
		return "", err
	}
	m := regexp.MustCompile(p.pattern).FindStringSubmatch(string(content))
	if m == nil {
		return "", fmt.Errorf("%s: no pin matching %s in %s", p.name, p.pattern, p.file)
	}
	return m[1], nil
}

func differ(pinned, latest string) bool {
	return strings.TrimPrefix(pinned, "v") != strings.TrimPrefix(latest, "v")
}

func majorBehind(pinned, latest string) bool {
	major, _, _ := strings.Cut(strings.TrimPrefix(latest, "v"), ".")
	return strings.TrimPrefix(pinned, "v") != major
}

const requestTimeout = 30 * time.Second

func getJSON(ctx context.Context, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" && strings.HasPrefix(url, "https://api.github.com/") {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: requestTimeout}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET %s: %s %s", url, resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func githubRelease(repo string) func(context.Context, string) (string, error) {
	return func(ctx context.Context, _ string) (string, error) {
		var release struct {
			TagName string `json:"tag_name"`
		}
		err := getJSON(ctx, "https://api.github.com/repos/"+repo+"/releases/latest", &release)
		return strings.TrimPrefix(release.TagName, "v"), err
	}
}

func npmLatest(name string) func(context.Context, string) (string, error) {
	return func(ctx context.Context, _ string) (string, error) {
		var pkg struct {
			Version string `json:"version"`
		}
		err := getJSON(ctx, "https://registry.npmjs.org/"+name+"/latest", &pkg)
		return pkg.Version, err
	}
}

func pypiLatest(name string) func(context.Context, string) (string, error) {
	return func(ctx context.Context, _ string) (string, error) {
		var project struct {
			Info struct {
				Version string `json:"version"`
			} `json:"info"`
		}
		err := getJSON(ctx, "https://pypi.org/pypi/"+name+"/json", &project)
		return project.Info.Version, err
	}
}

func dockerTag(repo, reference string) func(context.Context, string) (string, error) {
	return func(ctx context.Context, pinned string) (string, error) {
		digest := func(tag string) (string, error) {
			var t struct {
				Digest string `json:"digest"`
			}
			err := getJSON(ctx, "https://hub.docker.com/v2/repositories/library/"+repo+"/tags/"+tag, &t)
			return t.Digest, err
		}
		want, err := digest(reference)
		if err != nil {
			return "", err
		}
		got, err := digest(pinned)
		if err != nil {
			return "", err
		}
		if got == want {
			return pinned, nil
		}
		return "newer than " + pinned + " (" + reference + ")", nil
	}
}

func goCommand(ctx context.Context, root string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return nil, fmt.Errorf("go %s: %s", strings.Join(args, " "), exitErr.Stderr)
	}
	return out, err
}

func goModules(ctx context.Context, root string, holds []string) ([]finding, error) {
	edit, err := goCommand(ctx, root, "mod", "edit", "-json")
	if err != nil {
		return nil, err
	}
	var mod struct{ Require []struct{ Path string } }
	if err := json.Unmarshal(edit, &mod); err != nil {
		return nil, err
	}
	args := []string{"list", "-m", "-u", "-json"}
	for _, r := range mod.Require {
		args = append(args, r.Path)
	}
	out, err := goCommand(ctx, root, args...)
	if err != nil {
		return nil, err
	}
	var findings []finding
	decoder := json.NewDecoder(strings.NewReader(string(out)))
	for decoder.More() {
		var m struct {
			Path     string
			Version  string
			Main     bool
			Indirect bool
			Update   *struct{ Version string }
		}
		if err := decoder.Decode(&m); err != nil {
			return nil, err
		}
		if m.Main || m.Update == nil {
			continue
		}
		status := "behind"
		if slices.Contains(holds, m.Path) || (m.Indirect && slices.Contains(holds, "indirect")) {
			status = "held"
		}
		findings = append(findings, finding{m.Path, m.Version, m.Update.Version, status})
	}
	return findings, nil
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
)

const (
	matrixFile  = "matrix.json"
	expectMade  = "created"
	expectNoDir = "refused"
)

type matrix struct {
	Commit       string            `json:"commit"`
	Created      time.Time         `json:"created"`
	ExcludeNewer string            `json:"excludeNewer"`
	PictSeed     int               `json:"pictSeed"`
	Django       map[string]string `json:"django"`
	Pythons      map[string]string `json:"pythons"`
	Projects     []project         `json:"projects"`
}

type project struct {
	Name      string         `json:"name"`
	Group     string         `json:"group"`
	Toolchain string         `json:"toolchain"`
	Node      string         `json:"node"`
	Expect    string         `json:"expect"`
	Answers   config.Answers `json:"answers"`
}

type run struct {
	dir    string
	matrix matrix
}

func (r run) answersPath(name string) string { return filepath.Join(r.dir, "answers", name+".json") }
func (r run) projectsDir() string            { return filepath.Join(r.dir, "projects") }
func (r run) projectDir(name string) string  { return filepath.Join(r.dir, "projects", name) }
func (r run) logDir(name string) string      { return filepath.Join(r.dir, "logs", name) }
func (r run) binary() string                 { return filepath.Join(r.dir, "bin", "django-stack") }

func (p project) usesDocker() bool { return p.Answers.Database != config.SQLite }

func (p project) usesNode() bool {
	return p.Answers.Stack == config.StackTailwind || p.Answers.Stack == config.StackTailwindHTMX
}

func (p project) hasAddOn(name string) bool {
	return slices.Contains(p.Answers.AddOns, name)
}

func openRun(dir string) (run, error) {
	if dir == "" {
		return run{}, errors.New("-run is required: the run folder written by select")
	}
	data, err := os.ReadFile(filepath.Join(dir, matrixFile))
	if err != nil {
		return run{}, fmt.Errorf("read the run's selection: %w", err)
	}
	var m matrix
	if err := json.Unmarshal(data, &m); err != nil {
		return run{}, fmt.Errorf("decode %s: %w", matrixFile, err)
	}
	return run{dir: dir, matrix: m}, nil
}

func (r run) only(names string) ([]project, error) {
	if names == "" {
		return r.matrix.Projects, nil
	}
	byName := make(map[string]project, len(r.matrix.Projects))
	for _, p := range r.matrix.Projects {
		byName[p.Name] = p
	}
	var picked []project
	for name := range strings.SplitSeq(names, ",") {
		p, ok := byName[strings.TrimSpace(name)]
		if !ok {
			return nil, fmt.Errorf("no project named %q in this run", name)
		}
		picked = append(picked, p)
	}
	return picked, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func defaultRunDir(ctx context.Context, label string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sha, err := gitOutput(ctx, ".", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("read the current commit: %w", err)
	}
	return filepath.Join(home, "django-stack-runs", label+"-"+sha), nil
}

func refuseInsideRepository(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	existing := abs
	for {
		if _, err := os.Stat(existing); err == nil {
			break
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return nil
		}
		existing = parent
	}
	for dir := existing; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return fmt.Errorf("refusing to write run output inside the git repository %s; use a folder such as ~/django-stack-runs", dir)
		}
		if filepath.Dir(dir) == dir {
			return nil
		}
	}
}

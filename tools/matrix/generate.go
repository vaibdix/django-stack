package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

type generateResult struct {
	Exit      int     `json:"exit"`
	Seconds   float64 `json:"seconds"`
	Race      bool    `json:"race"`
	DirExists bool    `json:"dirExists"`
	Expect    string  `json:"expect"`
	OK        bool    `json:"ok"`
}

func runGenerate(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	dir := fs.String("run", "", "run folder written by select")
	workers := fs.Int("workers", 6, "projects generated at once")
	only := fs.String("only", "", "comma-separated project names (default: all)")
	timeout := fs.Duration("timeout", 30*time.Minute, "limit per project")
	resume := fs.Bool("resume", false, "skip projects an earlier run of this command generated as expected")
	if err := fs.Parse(args); err != nil {
		return err
	}
	r, err := openRun(*dir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(r.binary()); err != nil {
		return fmt.Errorf("no binary in the run folder; run build first: %w", err)
	}
	projects, err := r.only(*only)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(r.projectsDir(), 0o755); err != nil {
		return err
	}
	shims, err := buildShims(filepath.Join(r.dir, "shims"))
	if err != nil {
		return err
	}

	var mu sync.Mutex
	failed := 0
	databaseLane := func(p project) string {
		if p.usesDocker() {
			return string(p.Answers.Database)
		}
		return ""
	}
	inLanes(projects, *workers, databaseLane, func(p project) {
		var previous generateResult
		if *resume && readJSON(filepath.Join(r.logDir(p.Name), "generate.json"), &previous) == nil && previous.OK {
			return
		}
		res := generateOne(ctx, r, p, shims, *timeout)
		mu.Lock()
		defer mu.Unlock()
		status := "ok"
		if !res.OK {
			status = "FAIL"
			failed++
		}
		_, _ = fmt.Fprintf(stdout, "%-4s %-28s exit=%d %6.1fs race=%t\n", status, p.Name, res.Exit, res.Seconds, res.Race)
	})
	_, _ = fmt.Fprintf(stdout, "generated %d projects, %d not as expected\n", len(projects), failed)
	return nil
}

func generateOne(ctx context.Context, r run, p project, shims map[string]string, timeout time.Duration) generateResult {
	logDir := r.logDir(p.Name)
	_ = os.MkdirAll(logDir, 0o755)
	target := r.projectDir(p.Name)
	if _, err := os.Stat(target); err == nil {
		stopDocker(ctx, target)
		_ = os.RemoveAll(target)
	}

	env := append(os.Environ(),
		"UV_EXCLUDE_NEWER="+r.matrix.ExcludeNewer,
		"PIP_UPLOADED_PRIOR_TO="+r.matrix.ExcludeNewer,
		"GORACE=halt_on_error=0",
	)
	if path, ok := shims[shimFor(p)]; ok {
		env = append(env, "PATH="+path)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var log bytes.Buffer
	cmd := exec.CommandContext(ctx, r.binary(), "-config", r.answersPath(p.Name))
	cmd.Dir = r.projectsDir()
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = &log, &log
	start := time.Now()
	err := cmd.Run()
	res := generateResult{Seconds: time.Since(start).Seconds(), Expect: p.Expect}
	var exit *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exit):
		res.Exit = exit.ExitCode()
	default:
		res.Exit = -1
		_, _ = fmt.Fprintf(&log, "\nmatrix: %v\n", err)
	}
	_ = os.WriteFile(filepath.Join(logDir, "generate.log"), log.Bytes(), 0o644)
	res.Race = bytes.Contains(log.Bytes(), []byte("WARNING: DATA RACE"))
	_, statErr := os.Stat(target)
	res.DirExists = statErr == nil
	if res.DirExists {
		stopDatabase(ctx, target)
	}
	if p.Expect == expectNoDir {
		res.OK = res.Exit == 1 && !res.DirExists && !res.Race
	} else {
		res.OK = res.Exit == 0 && res.DirExists && !res.Race
	}
	_ = writeJSON(filepath.Join(logDir, "generate.json"), res)
	return res
}

func shimFor(p project) string {
	var hidden []string
	if p.Toolchain == "pip" {
		hidden = append(hidden, "nouv")
	}
	if p.Node == "npm" && p.usesNode() {
		hidden = append(hidden, "nopnpm")
	}
	return strings.Join(hidden, "-")
}

func buildShims(root string) (map[string]string, error) {
	variants := map[string][]string{
		"nouv":        {"uv", "uvx"},
		"nopnpm":      {"pnpm"},
		"nouv-nopnpm": {"uv", "uvx", "pnpm"},
	}
	paths := map[string]string{}
	for name, hidden := range variants {
		dir := filepath.Join(root, name)
		if err := os.RemoveAll(dir); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		for _, pathDir := range filepath.SplitList(os.Getenv("PATH")) {
			entries, err := os.ReadDir(pathDir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if slices.Contains(hidden, e.Name()) {
					continue
				}
				link := filepath.Join(dir, e.Name())
				if _, err := os.Lstat(link); err == nil {
					continue
				}
				_ = os.Symlink(filepath.Join(pathDir, e.Name()), link)
			}
		}
		paths[name] = dir
	}
	return paths, nil
}

func inLanes(projects []project, workers int, lane func(project) string, fn func(project)) {
	shared := []project{}
	lanes := map[string][]project{}
	for _, p := range projects {
		if key := lane(p); key != "" {
			lanes[key] = append(lanes[key], p)
		} else {
			shared = append(shared, p)
		}
	}
	var wg sync.WaitGroup
	for _, queue := range lanes {
		wg.Go(func() { forEach(queue, 1, fn) })
	}
	wg.Go(func() { forEach(shared, workers, fn) })
	wg.Wait()
}

func forEach(projects []project, workers int, fn func(project)) {
	jobs := make(chan project)
	var wg sync.WaitGroup
	for range max(workers, 1) {
		wg.Go(func() {
			for p := range jobs {
				fn(p)
			}
		})
	}
	for _, p := range projects {
		jobs <- p
	}
	close(jobs)
	wg.Wait()
}

func stopDocker(ctx context.Context, projectDir string) {
	if _, err := os.Stat(filepath.Join(projectDir, "docker-compose.yml")); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "compose", "down", "-v", "--remove-orphans")
	cmd.Dir = projectDir
	_ = cmd.Run()
}

func stopDatabase(ctx context.Context, projectDir string) {
	if _, err := os.Stat(filepath.Join(projectDir, "docker-compose.yml")); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "compose", "down")
	cmd.Dir = projectDir
	_ = cmd.Run()
}

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runClean(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	dir := fs.String("run", "", "run folder written by select")
	if err := fs.Parse(args); err != nil {
		return err
	}
	r, err := openRun(*dir)
	if err != nil {
		return err
	}
	for _, p := range r.matrix.Projects {
		target := r.projectDir(p.Name)
		if _, err := os.Stat(target); err != nil {
			continue
		}
		stopDocker(ctx, target)
		for _, heavy := range []string{".venv", "node_modules", ".mypy_cache", ".ruff_cache"} {
			_ = os.RemoveAll(filepath.Join(target, heavy))
		}
	}
	_ = os.RemoveAll(filepath.Join(r.dir, "scratch"))
	_ = os.RemoveAll(filepath.Join(r.dir, "shims"))

	leftovers := dockerLeftovers(ctx, r)
	size, _ := exec.CommandContext(ctx, "du", "-sh", r.dir).Output()
	_, _ = fmt.Fprintf(stdout, "run folder: %s", size)
	if len(leftovers) > 0 {
		_, _ = fmt.Fprintf(stdout, "Docker objects from this run are still present:\n  %s\n", strings.Join(leftovers, "\n  "))
		return fmt.Errorf("%d Docker objects left behind", len(leftovers))
	}
	_, _ = fmt.Fprintln(stdout, "no containers or volumes from this run remain")
	return nil
}

func dockerLeftovers(ctx context.Context, r run) []string {
	names := map[string]bool{}
	for _, p := range r.matrix.Projects {
		names[strings.ToLower(strings.ReplaceAll(p.Name, ".", ""))] = true
	}
	var leftovers []string
	containers, _ := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", `{{.Label "com.docker.compose.project"}} {{.Names}}`).Output()
	for _, line := range strings.Split(strings.TrimSpace(string(containers)), "\n") {
		project, name, _ := strings.Cut(line, " ")
		if names[project] {
			leftovers = append(leftovers, "container "+name)
		}
	}
	volumes, _ := exec.CommandContext(ctx, "docker", "volume", "ls", "--format", `{{.Label "com.docker.compose.project"}} {{.Name}}`).Output()
	for _, line := range strings.Split(strings.TrimSpace(string(volumes)), "\n") {
		project, name, _ := strings.Cut(line, " ")
		if names[project] {
			leftovers = append(leftovers, "volume "+name)
		}
	}
	return leftovers
}

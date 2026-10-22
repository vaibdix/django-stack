package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func runClean(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("clean", flag.ContinueOnError)
	dir := flags.String("run", "", "run folder written by select")
	if err := flags.Parse(args); err != nil {
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
		removeBuildOutput(target)
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

func removeBuildOutput(target string) {
	root, err := os.OpenRoot(target)
	if err != nil {
		return
	}
	defer func() { _ = root.Close() }()
	var doomed []string
	_ = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == "." {
			return nil
		}
		if d.Name() != ".git" && slices.Contains(skipDirs, d.Name()) {
			doomed = append(doomed, path)
			return fs.SkipDir
		}
		return nil
	})
	for _, path := range doomed {
		_ = root.RemoveAll(path)
	}
}

func dockerLeftovers(ctx context.Context, r run) []string {
	names := map[string]bool{}
	for _, p := range r.matrix.Projects {
		names[strings.ToLower(strings.ReplaceAll(p.Name, ".", ""))] = true
	}
	var leftovers []string
	containers, _ := exec.CommandContext(ctx, "docker", "ps", "-a", "--format",
		`{{.Label "com.docker.compose.project"}} {{.Names}} {{.Label "com.docker.compose.project.working_dir"}}`).Output()
	for line := range strings.SplitSeq(strings.TrimSpace(string(containers)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || !names[fields[0]] {
			continue
		}
		if strings.HasPrefix(filepath.Clean(fields[2]), filepath.Clean(r.dir)+string(filepath.Separator)) {
			leftovers = append(leftovers, "container "+fields[1])
		} else {
			delete(names, fields[0])
		}
	}
	volumes, _ := exec.CommandContext(ctx, "docker", "volume", "ls", "--format", `{{.Label "com.docker.compose.project"}} {{.Name}}`).Output()
	for line := range strings.SplitSeq(strings.TrimSpace(string(volumes)), "\n") {
		project, name, _ := strings.Cut(line, " ")
		if names[project] {
			leftovers = append(leftovers, "volume "+name)
		}
	}
	return leftovers
}

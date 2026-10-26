//go:build !windows

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func installDirs() []string {
	dirs := []string{"/usr/local/bin"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "bin"))
	}
	return dirs
}

func install(out io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate the running executable: %w", err)
	}
	return installTo(out, exe, installDirs())
}

func installTo(out io.Writer, exe string, dirs []string) error {
	var attempts []string
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			attempts = append(attempts, fmt.Sprintf("%s (%v)", dir, err))
			continue
		}

		target := filepath.Join(dir, "django-stack")
		if err := copyExecutable(exe, target); err != nil {
			attempts = append(attempts, fmt.Sprintf("%s (%v)", dir, err))
			continue
		}

		_, _ = fmt.Fprintf(out, "Installed django-stack to %s\n", target)
		if !onPath(dir) {
			_, _ = fmt.Fprintf(out, "\n%s is not on your PATH. Add this to your shell profile:\n", dir)
			_, _ = fmt.Fprintf(out, "    export PATH=\"%s:$PATH\"\n", dir)
		}
		return nil
	}

	return fmt.Errorf("could not install to any of: %s\n\nTry re-running with sudo, or install with:\n    go install github.com/vaibdix/django-stack/cmd/django-stack@latest",
		strings.Join(attempts, ", "))
}

func onPath(dir string) bool {
	return slices.Contains(filepath.SplitList(os.Getenv("PATH")), dir)
}

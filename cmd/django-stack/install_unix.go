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

// installDirs are tried in order.
//
// /usr/local/bin is conventional but usually needs root. ~/.local/bin is on the
// default PATH of most modern distributions and needs no privileges, so it is
// the fallback rather than a hard failure — the previous version simply
// surfaced a permission error and gave up.
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
		return fmt.Errorf("failed to locate the running executable: %w", err)
	}

	payload, err := os.ReadFile(exe) //nolint:gosec // reading our own binary
	if err != nil {
		return fmt.Errorf("failed to read the running executable: %w", err)
	}

	var attempts []string
	for _, dir := range installDirs() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			attempts = append(attempts, fmt.Sprintf("%s (%v)", dir, err))
			continue
		}

		target := filepath.Join(dir, "django-stack")
		// 0755: this is an executable, so the exec bits are required.
		if err := os.WriteFile(target, payload, 0o755); err != nil { //nolint:gosec // must be executable
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

	return fmt.Errorf("could not install to any of: %s\n\nTry re-running with sudo, or install with:\n    go install github.com/vaibdix/django-cli/cmd/django-stack@latest",
		strings.Join(attempts, ", "))
}

// onPath reports whether dir is already listed in $PATH.
func onPath(dir string) bool {
	return slices.Contains(filepath.SplitList(os.Getenv("PATH")), dir)
}

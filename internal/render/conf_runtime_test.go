package render

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const confRuntimeTimeout = 3 * time.Minute

func uvRunPydantic(t *testing.T, dir, script string) (string, bool) {
	t.Helper()

	uv := requireUV(t)

	ctx, cancel := context.WithTimeout(context.Background(), confRuntimeTimeout)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, uv, "run", "--no-project",
		"--with", "pydantic-settings", "--with", "django", "python", "-c", script)
	cmd.Dir = dir
	cmd.Env = environmentWithout(os.Environ(),
		"SECRET_KEY", "DEBUG", "ALLOWED_HOSTS", "DB_NAME", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT")

	out, runErr := cmd.CombinedOutput()
	return string(out), runErr == nil
}

func environmentWithout(environment []string, names ...string) []string {
	drop := make(map[string]bool, len(names))
	for _, name := range names {
		drop[name] = true
	}

	kept := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if !drop[name] {
			kept = append(kept, entry)
		}
	}
	return kept
}

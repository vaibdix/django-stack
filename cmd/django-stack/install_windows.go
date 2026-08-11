//go:build windows

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// pathUpdateScript appends the install directory to the user's PATH.
//
// The directory arrives through an environment variable rather than being
// interpolated into the script text: this is the one place a shell is involved,
// and building a command line by concatenation is how quoting bugs — and worse
// — get in.
const pathUpdateScript = `
$dir = $env:DJANGO_STACK_INSTALL_DIR
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$dir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$dir", "User")
}
`

func install(out io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate the running executable: %w", err)
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return fmt.Errorf("LOCALAPPDATA is not set; cannot determine an install directory")
	}

	installDir := filepath.Join(localAppData, "DjangoStack")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", installDir, err)
	}

	payload, err := os.ReadFile(exe) //nolint:gosec // reading our own binary
	if err != nil {
		return fmt.Errorf("failed to read the running executable: %w", err)
	}

	target := filepath.Join(installDir, "django-stack.exe")
	if err := os.WriteFile(target, payload, 0o755); err != nil { //nolint:gosec // must be executable
		return fmt.Errorf("failed to write %s: %w", target, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", pathUpdateScript)
	cmd.Env = append(os.Environ(), "DJANGO_STACK_INSTALL_DIR="+installDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("installed to %s but could not update PATH: %w\nOutput: %s", target, err, output)
	}

	fmt.Fprintf(out, "Installed django-stack to %s\n", target)
	fmt.Fprintln(out, "PATH updated. Restart your terminal for it to take effect.")
	return nil
}

package pyenv

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestPackageInstallArgsAreTheBaseCommandPlusPackages(t *testing.T) {
	t.Parallel()

	projectPath := filepath.Join(t.TempDir(), "demo")
	packages := []string{"django==6.1.2", "ruff"}

	for _, toolchain := range []Toolchain{Pip, UV} {
		t.Run(toolchain.String(), func(t *testing.T) {
			t.Parallel()

			wantName, baseArgs := PackageInstallCommand(toolchain, projectPath)
			wantArgs := append(slices.Clone(baseArgs), packages...)

			name, args := PackageInstallArgs(toolchain, projectPath, packages...)
			if name != wantName {
				t.Fatalf("command = %q, want %q", name, wantName)
			}
			if !slices.Equal(args, wantArgs) {
				t.Fatalf("args = %#v, want %#v", args, wantArgs)
			}
		})
	}
}

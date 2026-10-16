//go:build !windows

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallReplacesTheTargetAtomically(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	exe := filepath.Join(dir, "built-django-stack")
	if err := os.WriteFile(exe, []byte("new build"), 0o755); err != nil {
		t.Fatalf("writing the fake binary: %v", err)
	}
	binDir := filepath.Join(dir, "bin")
	target := filepath.Join(binDir, "django-stack")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("creating bin: %v", err)
	}
	if err := os.WriteFile(target, []byte("old build"), 0o755); err != nil {
		t.Fatalf("writing the old install: %v", err)
	}
	before, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	var out bytes.Buffer
	if err := installTo(&out, exe, []string{binDir}); err != nil {
		t.Fatalf("installTo: %v", err)
	}

	after, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if os.SameFile(before, after) {
		t.Error("the old binary was overwritten in place; a running copy or macOS's signature cache sees a truncated file")
	}
	if got, _ := os.ReadFile(target); string(got) != "new build" {
		t.Errorf("installed content = %q", got)
	}
	if after.Mode().Perm()&0o111 == 0 {
		t.Errorf("the installed binary is not executable: %v", after.Mode())
	}
	leftovers, _ := filepath.Glob(filepath.Join(binDir, ".django-stack.*"))
	if len(leftovers) > 0 {
		t.Errorf("temporary files were left behind: %v", leftovers)
	}
}

func TestInstallingTheInstalledBinaryOverItselfKeepsIt(t *testing.T) {
	t.Parallel()

	binDir := t.TempDir()
	target := filepath.Join(binDir, "django-stack")
	if err := os.WriteFile(target, []byte("current"), 0o755); err != nil {
		t.Fatalf("writing the install: %v", err)
	}

	var out bytes.Buffer
	if err := installTo(&out, target, []string{binDir}); err != nil {
		t.Fatalf("installTo: %v", err)
	}
	if got, _ := os.ReadFile(target); string(got) != "current" {
		t.Errorf("installing onto itself destroyed the binary: %q", got)
	}
}

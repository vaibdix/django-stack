package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func copyExecutable(source, target string) error {
	if same, err := sameFile(source, target); err == nil && same {
		return nil
	}

	in, err := os.Open(source) //nolint:gosec // copying our own binary
	if err != nil {
		return fmt.Errorf("failed to read the running executable: %w", err)
	}
	defer func() { _ = in.Close() }()

	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, target)
}

func sameFile(left, right string) (bool, error) {
	a, err := os.Stat(left)
	if err != nil {
		return false, err
	}
	b, err := os.Stat(right)
	if err != nil {
		return false, err
	}
	return os.SameFile(a, b), nil
}

func mergePathList(existing, dir string, separator string) (string, bool) {
	clean := func(entry string) string {
		return strings.ToLower(strings.TrimRight(filepath.Clean(strings.TrimSpace(entry)), `\/`))
	}
	want := clean(dir)
	for entry := range strings.SplitSeq(existing, separator) {
		if strings.TrimSpace(entry) != "" && clean(entry) == want {
			return existing, false
		}
	}
	if strings.TrimSpace(existing) == "" {
		return dir, true
	}
	return strings.TrimRight(existing, separator) + separator + dir, true
}

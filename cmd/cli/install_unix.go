//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func install() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	installDir := "/usr/local/bin"
	targetPath := filepath.Join(installDir, "django-stack")

	input, err := os.ReadFile(exe)
	if err != nil {
		return fmt.Errorf("failed to read executable: %v", err)
	}

	if err := os.WriteFile(targetPath, input, 0755); err != nil {
		return fmt.Errorf("failed to copy executable: %v", err)
	}

	fmt.Println("\nDjango Stack CLI installed successfully!")
	fmt.Println("You can now use 'django-stack' from any directory.")
	fmt.Printf("Installation directory: %s\n", installDir)

	return nil
}

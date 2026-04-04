//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func install() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return fmt.Errorf("LOCALAPPDATA environment variable not found")
	}

	installDir := filepath.Join(localAppData, "DjangoStack")
	targetPath := filepath.Join(installDir, "django-stack.exe")

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory: %v", err)
	}

	input, err := os.ReadFile(exe)
	if err != nil {
		return fmt.Errorf("failed to read executable: %v", err)
	}

	if err := os.WriteFile(targetPath, input, 0755); err != nil {
		return fmt.Errorf("failed to copy executable: %v", err)
	}

	cmd := exec.Command("powershell", "-Command",
		`$userPath = [Environment]::GetEnvironmentVariable("Path", "User"); if ($userPath -notlike "*`+installDir+`*") { [Environment]::SetEnvironmentVariable("Path", "$userPath;`+installDir+`", "User") }`)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update PATH: %v", err)
	}

	fmt.Println("\nDjango Stack CLI installed successfully!")
	fmt.Println("You can now use 'django-stack' from any directory.")
	fmt.Printf("Installation directory: %s\n", installDir)
	fmt.Println("\nPlease restart your terminal for the PATH changes to take effect.")

	return nil
}

//go:build windows

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	hwndBroadcast    = 0xffff
	wmSettingChange  = 0x001A
	smtoAbortIfHung  = 0x0002
	broadcastTimeout = 5000
)

func install(out io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate the running executable: %w", err)
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return errors.New("LOCALAPPDATA is not set; cannot determine an install directory")
	}

	installDir := filepath.Join(localAppData, "DjangoStack")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", installDir, err)
	}

	target := filepath.Join(installDir, "django-stack.exe")
	if err := copyExecutable(exe, target); err != nil {
		return fmt.Errorf("failed to write %s (close any running django-stack and retry): %w", target, err)
	}
	_, _ = fmt.Fprintf(out, "Installed django-stack to %s\n", target)

	changed, err := addToUserPath(installDir)
	if err != nil {
		return fmt.Errorf("installed to %s but could not update PATH: %w", target, err)
	}
	if changed {
		_, _ = fmt.Fprintln(out, "PATH updated. Open a new terminal for it to take effect.")
	}
	return nil
}

func addToUserPath(dir string) (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer func() { _ = key.Close() }()

	current, valueType, err := key.GetStringValue("Path")
	if errors.Is(err, registry.ErrNotExist) {
		current, valueType = "", registry.EXPAND_SZ
	} else if err != nil {
		return false, err
	}

	merged, changed := mergePathList(current, dir, ";")
	if !changed {
		return false, nil
	}

	if valueType == registry.SZ {
		err = key.SetStringValue("Path", merged)
	} else {
		err = key.SetExpandStringValue("Path", merged)
	}
	if err != nil {
		return false, err
	}
	broadcastEnvironmentChange()
	return true, nil
}

func broadcastEnvironmentChange() {
	environment, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	proc := windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")
	_, _, _ = proc.Call(hwndBroadcast, wmSettingChange, 0,
		uintptr(unsafe.Pointer(environment)), smtoAbortIfHung, broadcastTimeout, 0) //nolint:gosec // Win32 takes the string as a pointer
}

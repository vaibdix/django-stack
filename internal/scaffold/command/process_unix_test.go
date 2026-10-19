//go:build !windows

package command

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCancellingKillsTheWholeProcessTreeAndReturnsPromptly(t *testing.T) {
	t.Parallel()

	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	script := "sleep 60 & echo $! > " + pidFile + "; wait"

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			if _, err := os.Stat(pidFile); err == nil {
				time.Sleep(50 * time.Millisecond)
				cancel()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	start := time.Now()
	_, err := NewOS().Run(ctx, "", "sh", "-c", script)
	if err == nil {
		t.Fatal("a cancelled command reported success")
	}
	if elapsed := time.Since(start); elapsed > waitDelay+2*time.Second {
		t.Errorf("Run took %s to return after cancellation", elapsed)
	}

	raw, readErr := os.ReadFile(pidFile)
	if readErr != nil {
		t.Fatalf("reading the grandchild pid: %v", readErr)
	}
	pid, convErr := strconv.Atoi(strings.TrimSpace(string(raw)))
	if convErr != nil {
		t.Fatalf("parsing the grandchild pid %q: %v", raw, convErr)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !running(pid) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Errorf("the grandchild %d outlived the cancelled command", pid)
}

// running reports whether pid is still executing. A killed process whose
// parent never reaps it stays a zombie, which kill(pid, 0) still finds: in a
// container whose PID 1 is not an init (act's runner), every orphan does.
func running(pid int) bool {
	if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
		return false
	}
	stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return true // no procfs (macOS); kill(pid, 0) succeeding is the answer
	}
	// The state follows the parenthesised command name: "pid (comm) S ...".
	fields := strings.Fields(string(stat[strings.LastIndexByte(string(stat), ')')+1:]))
	return len(fields) == 0 || fields[0] != "Z"
}

package command_test

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-cli/internal/scaffold/command"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
)

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX shell utilities")
	}
}

func TestOSRunReturnsCombinedOutput(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	out, err := command.NewOS().Run(context.Background(), "", "echo", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "hello" {
		t.Errorf("output = %q, want %q", got, "hello")
	}
}

func TestOSRunUsesTheGivenDirectory(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	dir := t.TempDir()

	out, err := command.NewOS().Run(context.Background(), dir, "pwd")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := strings.TrimSpace(string(out))
	if !strings.HasSuffix(got, strings.TrimPrefix(dir, "/private")) {
		t.Errorf("ran in %q, want %q", got, dir)
	}
}

func TestOSRunErrorCarriesOutputAndCommandLine(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	_, err := command.NewOS().Run(context.Background(), "", "sh", "-c", "echo boom >&2; exit 3")
	if err == nil {
		t.Fatal("Run returned nil error for a command that exited 3")
	}

	var cmdErr *command.Error
	if !errors.As(err, &cmdErr) {
		t.Fatalf("error is %T, want *command.Error", err)
	}
	if cmdErr.Name != "sh" {
		t.Errorf("Name = %q, want %q", cmdErr.Name, "sh")
	}
	if !strings.Contains(string(cmdErr.Output), "boom") {
		t.Errorf("Output = %q, want it to contain %q", cmdErr.Output, "boom")
	}
	if !strings.Contains(err.Error(), "sh -c") {
		t.Errorf("error text %q does not name the command line", err.Error())
	}
}

func TestOSRunHonoursContextCancellation(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := command.NewOS().Run(ctx, "", "sleep", "10")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Run returned nil error for a cancelled command")
	}
	if elapsed > 5*time.Second {
		t.Errorf("Run took %v; the context deadline did not stop the child", elapsed)
	}
}

func TestFakeRecordsCallsWithoutExecuting(t *testing.T) {
	t.Parallel()

	f := commandtest.NewFake()

	if _, err := f.Run(context.Background(), "/proj", "git", "init"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := f.Run(context.Background(), "/proj", "python", "-m", "venv", ".venv"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{"git init", "python -m venv .venv"}
	got := f.CommandLines()
	if len(got) != len(want) {
		t.Fatalf("recorded %d calls %q, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("call %d = %q, want %q", i, got[i], want[i])
		}
	}
	if !f.Ran("git init") {
		t.Error("Ran(\"git init\") = false")
	}
	if f.Ran("git status") {
		t.Error("Ran reported a command that was never run")
	}
	if dir := f.Calls()[0].Dir; dir != "/proj" {
		t.Errorf("Dir = %q, want %q", dir, "/proj")
	}
}

func TestFakeReturnsConfiguredOutputAndErrors(t *testing.T) {
	t.Parallel()

	f := commandtest.NewFake()
	f.Responses["python --version"] = []byte("Python 3.12.1\n")
	f.FailWith("uv venv .venv", "uv exploded")

	out, err := f.Run(context.Background(), "", "python", "--version")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(string(out), "3.12.1") {
		t.Errorf("output = %q, want the configured response", out)
	}

	if _, err := f.Run(context.Background(), "", "uv", "venv", ".venv"); err == nil {
		t.Fatal("expected the configured failure")
	} else if !strings.Contains(err.Error(), "uv exploded") {
		t.Errorf("error = %q, want it to mention the configured message", err)
	}
}

func TestFakeIsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()

	f := commandtest.NewFake()

	done := make(chan struct{})
	for range 4 {
		go func() {
			defer func() { done <- struct{}{} }()
			for range 50 {
				_, _ = f.Run(context.Background(), "", "echo", "hi")
				_ = f.CommandLines()
			}
		}()
	}
	for range 4 {
		<-done
	}

	if got := len(f.Calls()); got != 200 {
		t.Errorf("recorded %d calls, want 200", got)
	}
}

func TestErrorsNeverEchoSecretArguments(t *testing.T) {
	t.Parallel()

	err := &command.Error{
		Name: "docker",
		Args: []string{"run", "-e", "MYSQL_ROOT_PASSWORD=hunter2", "-e", "SENTRY_DSN=https://k@x/1", "-e", "MYSQL_DATABASE=shop"},
		Err:  errors.New("exit status 125"),
	}
	msg := err.Error()
	for _, secret := range []string{"hunter2", "https://k@x/1"} {
		if strings.Contains(msg, secret) {
			t.Errorf("the error shown in the TUI leaks %q: %s", secret, msg)
		}
	}
	if !strings.Contains(msg, "MYSQL_DATABASE=shop") {
		t.Errorf("a harmless argument was redacted too: %s", msg)
	}
}

func TestChildProcessesAreAskedForPlainOutput(t *testing.T) {
	skipOnWindows(t)
	t.Setenv("FORCE_COLOR", "3")
	t.Setenv("CLICOLOR_FORCE", "1")
	t.Setenv("NO_COLOR", "")

	out, err := command.NewOS().Run(context.Background(), "", "sh", "-c",
		`printf '%s|%s|%s' "${FORCE_COLOR-unset}" "${CLICOLOR_FORCE-unset}" "$NO_COLOR"`)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := string(out); got != "unset|unset|1" {
		t.Errorf("child saw FORCE_COLOR|CLICOLOR_FORCE|NO_COLOR = %q, want unset|unset|1: "+
			"forced colour turns a freeze like django==6.1.2 into escape codes that do not parse", got)
	}
}

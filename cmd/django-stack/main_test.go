package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/scaffold"
	"github.com/vaibdix/django-cli/internal/tui"
)

func TestVersionFlagPrintsBuildInfo(t *testing.T) {
	version, commit, date = "1.2.3", "abc1234", "2026-08-10T00:00:00Z"
	t.Cleanup(func() { version, commit, date = "dev", "", "" })

	var stdout, stderr bytes.Buffer
	if err := run([]string{"-version"}, &stdout, &stderr); err != nil {
		t.Fatalf("run(-version) = %v, want nil", err)
	}

	out := stdout.String()
	for _, want := range []string{"1.2.3", "abc1234", "2026-08-10"} {
		if !strings.Contains(out, want) {
			t.Errorf("version output %q is missing %q", out, want)
		}
	}
}

func TestVersionFlagOmitsEmptyFields(t *testing.T) {
	version, commit, date = "dev", "", ""

	var stdout, stderr bytes.Buffer
	if err := run([]string{"-version"}, &stdout, &stderr); err != nil {
		t.Fatalf("run(-version) = %v, want nil", err)
	}

	out := stdout.String()
	if strings.Contains(out, "commit:") || strings.Contains(out, "date:") {
		t.Errorf("version output %q includes empty commit/date lines", out)
	}
}

func TestHelpFlagReturnsErrHelp(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	err := run([]string{"-h"}, &stdout, &stderr)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("run(-h) = %v, want flag.ErrHelp", err)
	}
	if !strings.Contains(stderr.String(), "django-stack") {
		t.Errorf("help output does not describe the command: %q", stderr.String())
	}
}

func TestUnknownFlagIsRejected(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	err := run([]string{"-nope"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run(-nope) = nil; an unknown flag must be an error")
	}
	if !strings.Contains(stderr.String(), "not defined") {
		t.Errorf("stderr %q does not explain the unknown flag", stderr.String())
	}
}

func TestUnexpectedPositionalArgumentIsRejected(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	err := run([]string{"unexpected"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run(\"unexpected\") = nil; a stray argument must be an error")
	}
	if !strings.Contains(err.Error(), "unexpected") {
		t.Errorf("error %q does not name the offending argument", err)
	}
}

func TestExtraArgumentAfterFlagIsRejected(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	if err := run([]string{"-version", "stray"}, &stdout, &stderr); err == nil {
		t.Fatal("a trailing argument after -version was accepted")
	}
}

func TestGoInstallBuildsReportTheModuleVersion(t *testing.T) {
	original := readBuildInfo
	version, commit, date = "dev", "", ""
	t.Cleanup(func() {
		readBuildInfo = original
		version, commit, date = "dev", "", ""
	})
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v1.4.0"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "0123456789abcdef"},
				{Key: "vcs.time", Value: "2026-10-16T08:00:00Z"},
			},
		}, true
	}
	fillVersionFromBuildInfo()

	var stdout, stderr bytes.Buffer
	if err := run([]string{"-version"}, &stdout, &stderr); err != nil {
		t.Fatalf("run(-version) = %v", err)
	}
	for _, want := range []string{"v1.4.0", "0123456", "2026-10-16"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("version output %q is missing %q", stdout.String(), want)
		}
	}
}

func TestLinkerFlagsWinOverBuildInfo(t *testing.T) {
	original := readBuildInfo
	version, commit, date = "1.2.3", "abc1234", "2026-08-10T00:00:00Z"
	t.Cleanup(func() {
		readBuildInfo = original
		version, commit, date = "dev", "", ""
	})
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}}, true
	}
	fillVersionFromBuildInfo()

	var stdout, stderr bytes.Buffer
	if err := run([]string{"-version"}, &stdout, &stderr); err != nil {
		t.Fatalf("run(-version) = %v", err)
	}
	if strings.Contains(stdout.String(), "v9.9.9") {
		t.Errorf("build info overrode the version set by the release build: %q", stdout.String())
	}
}

func TestEveryOutcomeHasItsOwnExitStatus(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		err  error
		want int
	}{
		"success":             {nil, exitOK},
		"help":                {flag.ErrHelp, exitOK},
		"failure":             {errors.New("boom"), exitFailed},
		"failed verification": {fmt.Errorf("setup: %w", &scaffold.VerificationError{Check: "test suite"}), exitVerificationFailed},
		"cancelled":           {fmt.Errorf("run cancelled before x: %w", context.Canceled), exitCancelled},
	}
	for name, tc := range cases {
		if got := exitCode(tc.err); got != tc.want {
			t.Errorf("%s: exitCode(%v) = %d, want %d", name, tc.err, got, tc.want)
		}
	}
}

func TestATerminalFailureIsReturnedRatherThanSwallowed(t *testing.T) {
	original := runProgram
	t.Cleanup(func() { runProgram = original })
	runProgram = func(*tui.Model) error { return errors.New("no tty") }

	err := runTUI()
	if err == nil || !strings.Contains(err.Error(), "no tty") {
		t.Fatalf("runTUI() = %v, want the terminal error", err)
	}
	if exitCode(err) != exitFailed {
		t.Errorf("a terminal failure exits %d, want %d", exitCode(err), exitFailed)
	}
}

func TestQuittingBeforeGenerationExitsZero(t *testing.T) {
	original := runProgram
	t.Cleanup(func() { runProgram = original })
	runProgram = func(*tui.Model) error { return nil }

	if err := runTUI(); err != nil {
		t.Fatalf("runTUI() = %v, want nil when no project was generated", err)
	}
}

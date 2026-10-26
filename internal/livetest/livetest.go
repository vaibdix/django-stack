package livetest

import (
	"os"
	"testing"
)

const Env = "DJANGO_STACK_LIVE"

func Enabled() bool { return os.Getenv(Env) != "" }

func Require(t testing.TB, reason string) {
	t.Helper()
	if !Enabled() {
		t.Skipf("set %s=1 to %s", Env, reason)
	}
}

func MissingTool(t testing.TB, format string, args ...any) {
	t.Helper()
	if os.Getenv("CI") == "true" {
		t.Fatalf(format, args...)
	}
	t.Skipf(format, args...)
}

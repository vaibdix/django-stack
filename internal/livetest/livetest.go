package livetest

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
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

var allocated = struct {
	sync.Mutex
	ports map[int]bool
	names int
}{ports: map[int]bool{}}

func Port(t testing.TB) string {
	t.Helper()
	allocated.Lock()
	defer allocated.Unlock()
	for range 50 {
		var lc net.ListenConfig
		listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("finding a free port: %v", err)
		}
		addr, ok := listener.Addr().(*net.TCPAddr)
		_ = listener.Close()
		if !ok || allocated.ports[addr.Port] || !freeEverywhere(t.Context(), addr.Port) {
			continue
		}
		allocated.ports[addr.Port] = true
		return strconv.Itoa(addr.Port)
	}
	t.Fatal("no free TCP port after 50 tries")
	return ""
}

func freeEverywhere(ctx context.Context, port int) bool {
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", net.JoinHostPort("0.0.0.0", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

func Name(t testing.TB, prefix string) string {
	t.Helper()
	allocated.Lock()
	defer allocated.Unlock()
	allocated.names++
	return fmt.Sprintf("%s-%d-%d", prefix, os.Getpid(), allocated.names)
}

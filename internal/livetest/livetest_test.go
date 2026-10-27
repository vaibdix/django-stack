package livetest

import (
	"net"
	"strings"
	"sync"
	"testing"
)

func TestPortsAreFreeAndNeverHandedOutTwice(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	seen := map[string]bool{}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			port := Port(t)
			mu.Lock()
			defer mu.Unlock()
			if seen[port] {
				t.Errorf("port %s was handed out twice", port)
			}
			seen[port] = true
		})
	}
	wg.Wait()
	for port := range seen {
		var lc net.ListenConfig
		listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:"+port)
		if err != nil {
			t.Errorf("port %s is not free: %v", port, err)
			continue
		}
		_ = listener.Close()
	}
}

func TestNamesAreUniqueAndKeepTheirPrefix(t *testing.T) {
	t.Parallel()

	first, second := Name(t, "djstacklive-postgres"), Name(t, "djstacklive-postgres")
	if first == second {
		t.Errorf("two calls returned %s", first)
	}
	if !strings.HasPrefix(first, "djstacklive-postgres-") {
		t.Errorf("Name dropped its prefix: %s", first)
	}
}

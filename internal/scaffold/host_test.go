package scaffold

import (
	"errors"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
)

func TestTheHostProbeChecksEveryNeedInTheCatalog(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.Responses["node --version"] = []byte("v18.20.0\n")
	fake.Responses["npm --version"] = []byte("10.8.0\n")
	onPath := map[string]bool{"node": true, "npm": true}
	d := deps{lookPath: func(name string) (string, error) {
		if onPath[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}}.withDefaults()

	host := probeHost(t.Context(), fake, d)
	if err := host["node>=20"]; err == nil || !strings.Contains(err.Error(), "Node.js 20") {
		t.Errorf("Node 18 passed the node>=20 need: %v", err)
	}
	if err := host["npm|pnpm"]; err != nil {
		t.Errorf("npm on PATH did not satisfy npm|pnpm: %v", err)
	}
	if err := host["git"]; err == nil {
		t.Error("git is missing but the probe found it")
	}
	if _, probed := host["docker"]; probed {
		t.Error("docker has its own step in the setup and is not probed here")
	}
	if err := host.Missing(config.StackNeeds(config.StackTailwind)); err == nil {
		t.Error("a Tailwind stack was offered on a machine with Node 18")
	}
	if err := host.Missing(config.StackNeeds(config.StackStandard)); err != nil {
		t.Errorf("the standard stack needs nothing from the host, got %v", err)
	}
}

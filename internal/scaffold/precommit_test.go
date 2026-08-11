package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func TestDjangoTrackDerivesTheSeriesFromTheSelectedVersion(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"6.1":    "6.1",
		"6.1.2":  "6.1",
		"6.0.3":  "6.0",
		"5.2.12": "5.2",
		"4.2":    "4.2",
	}

	for version, want := range cases {
		e := &Env{Cfg: config.Project{DjangoVersion: version}}
		if got := e.djangoTrack(); got != want {
			t.Errorf("djangoTrack() for %q = %q, want %q", version, got, want)
		}
	}
}

func TestDefaultDjangoTrackIsNotStale(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: config.Project{}}
	got := e.djangoTrack()

	if got != defaultDjangoTrack {
		t.Fatalf("an unset version should fall back to %q, got %q", defaultDjangoTrack, got)
	}

	config := e.generatePreCommitConfig()
	if !strings.Contains(config, `--target-version, "`+defaultDjangoTrack+`"`) {
		t.Errorf("the generated pre-commit config does not target %q:\n%s", defaultDjangoTrack, config)
	}

	if defaultDjangoTrack < "6.1" {
		t.Errorf("defaultDjangoTrack is %q; django-upgrade will decline every fix newer than that, "+
			"which is what four releases of drift looked like last time", defaultDjangoTrack)
	}
}

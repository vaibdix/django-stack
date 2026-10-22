package scaffold

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
)

func reviewClock() time.Time { return time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC) }

func TestTheEngineRefusesADjangoReleaseThePythonCannotRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		django, python string
		ok             bool
	}{
		{"6.1.2", "Python 3.11.9", false},
		{"6.0.9", "Python 3.10.18", false},
		{"6.1.2", "Python 3.12.11", true},
		{"5.2.18", "Python 3.10.18", true},
	}
	for _, c := range cases {
		cfg := config.Project{
			ProjectName: "demo", DjangoVersion: c.django, DatabaseType: config.SQLite,
			SelectedPythonPath: "/usr/bin/python3", SelectedPythonVersion: c.python,
		}
		_, err := Preflight(context.Background(), cfg, command.NewOS(), reviewClock())
		if c.ok && err != nil {
			t.Errorf("Django %s on %s was refused: %v", c.django, c.python, err)
		}
		if !c.ok && (err == nil || !strings.Contains(err.Error(), "does not support Python")) {
			t.Errorf("Django %s on %s: err = %v, want a refusal", c.django, c.python, err)
		}
	}
}

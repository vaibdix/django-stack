package scaffold

import (
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestAnUnparseableInterpreterVersionIsNotGuessed(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), Resolved: config.Resolved{PythonVersion: "unknown"}}
	if err := e.at(t.TempDir()).writePythonVersionFile(); err == nil {
		t.Error(".python-version was written from a guess instead of the selected interpreter")
	}
}

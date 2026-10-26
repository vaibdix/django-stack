package scaffold

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/vaibdix/django-stack/internal/assets"
)

func TestMain(m *testing.M) {
	code := m.Run()
	if run := flag.Lookup("test.run"); code == 0 && run != nil && run.Value.String() == "" {
		if missing := assets.Unrendered(); len(missing) > 0 {
			fmt.Fprintln(os.Stderr, "templates that no test renders:", missing)
			code = 1
		}
	}
	os.Exit(code)
}

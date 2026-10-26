package catalog

import (
	"strings"
	"testing"
	"time"
)

func TestTheEmbeddedCatalogIsValid(t *testing.T) {
	t.Parallel()

	c, err := Parse(catalogTOML)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(c.Django) == 0 || len(c.Stacks) == 0 || len(c.Databases) == 0 || len(c.AddOns) == 0 ||
		len(c.InstalledModules) == 0 || len(c.ScaffoldNames) == 0 {
		t.Errorf("a table is empty: %+v", c)
	}
	if got := c.Django[0].SecurityUntil(); !got.Equal(time.Date(2028, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Django %s support ends %v; a series is supported through the whole of its last month", c.Django[0].Series, got)
	}
}

func TestABrokenCatalogIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]struct{ toml, want string }{
		"unknown key":        {"[[stack]]\nname = \"a\"\ncolour = \"red\"\n", "colour"},
		"bad month":          {"[[django]]\nseries = \"9.0\"\npython = [12, 14]\nsupported_through = \"2027\"\n", "year and month"},
		"reversed python":    {"[[django]]\nseries = \"9.0\"\npython = [14, 12]\nsupported_through = \"2027-01\"\n", "reversed"},
		"duplicate":          {"[[stack]]\nname = \"a\"\n[[stack]]\nname = \"a\"\n", "listed twice"},
		"unknown requires":   {"[[addon]]\nname = \"a\"\nrequires = [\"b\"]\n", "not an add-on"},
		"one-sided conflict": {"[[addon]]\nname = \"a\"\nconflicts = [\"b\"]\n[[addon]]\nname = \"b\"\n", "does not list a"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse([]byte(c.toml)); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("Parse = %v, want an error mentioning %q", err, c.want)
			}
		})
	}
}

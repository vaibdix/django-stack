package scaffold

import (
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestEntriesArePlacedByTheirAnchorAndRankWhateverOrderTheyArrive(t *testing.T) {
	t.Parallel()

	base := []string{"a", "b", "c"}
	entries := []entry{
		{name: "late", after: atEnd, rank: 10},
		{name: "y", after: "a", rank: 20},
		{name: "x", after: "a", rank: 10},
		{name: "x2", after: "x"},
	}
	want := []string{"a", "x", "x2", "y", "b", "c", "late"}
	for range 2 {
		got, err := placeEntries("app", base, entries)
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("placeEntries = %v, %v; want %v", got, err, want)
		}
		slices.Reverse(entries)
	}
}

func TestPlacementRefusesClashesAndMissingAnchors(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		entries []entry
		want    string
	}{
		"same place":     {[]entry{{name: "x", after: "a", rank: 1}, {name: "y", after: "a", rank: 1}}, "both claim place 1"},
		"listed twice":   {[]entry{{name: "a", after: atEnd}}, "listed twice"},
		"missing anchor": {[]entry{{name: "x", after: "nowhere"}}, "not in the list"},
	}
	for name, c := range cases {
		if _, err := placeEntries("app", []string{"a"}, c.entries); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one mentioning %q", name, err, c.want)
		}
	}
	if _, err := orderBlocks("settings", []block{{rank: 3, text: "A = 1"}, {rank: 3, text: "B = 2"}}); err == nil {
		t.Error("two settings blocks in one place were accepted")
	}
	if _, err := orderRoutes([]route{{rank: 5, line: "a"}, {rank: 5, line: "b"}}); err == nil {
		t.Error("two routes in one place were accepted")
	}
}

func TestEveryFeatureCombinationPlacesWithoutAClash(t *testing.T) {
	t.Parallel()

	for _, django := range []string{"5.2.18", "6.1.2"} {
		for _, stack := range config.AllStacks() {
			for _, api := range []config.AddOn{config.RestFramework, config.Ninja} {
				cfg := config.Choices{Django: django, Stack: stack, AddOns: config.NewSet(api, config.AllAuth, config.Structlog, config.Sentry, config.Mypy, config.MultiEnv)}
				p := project{cfg: mustSpec(cfg)}
				if _, err := p.renderSettingsModule(); err != nil {
					t.Errorf("%s %s %s: settings: %v", django, stack, api, err)
				}
				if _, err := p.buildProjectURLsContent(); err != nil {
					t.Errorf("%s %s %s: urls: %v", django, stack, api, err)
				}
			}
		}
	}
}

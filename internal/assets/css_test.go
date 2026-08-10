package assets_test

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/assets"
)

// The CSS is embedded at build time via //go:embed and loaded in init(). If the
// embedded files are renamed or moved, the loader silently swallows the error
// and every generated project ships an empty stylesheet. These tests turn that
// silent failure into a build failure.

func TestBaseCSSIsEmbeddedAndNonEmpty(t *testing.T) {
	t.Parallel()

	css := assets.BaseCSS()

	if strings.TrimSpace(css) == "" {
		t.Fatal("base.css is empty; the //go:embed path in loader.go no longer resolves")
	}
	if !strings.Contains(css, "{") {
		t.Errorf("base.css does not look like CSS: %q", truncate(css))
	}
}

func TestHomeCSSIsEmbeddedAndNonEmpty(t *testing.T) {
	t.Parallel()

	css := assets.HomeCSS()

	if strings.TrimSpace(css) == "" {
		t.Fatal("home.css is empty; the //go:embed path in loader.go no longer resolves")
	}
	if !strings.Contains(css, "{") {
		t.Errorf("home.css does not look like CSS: %q", truncate(css))
	}
}

func TestBaseAndHomeCSSAreDistinct(t *testing.T) {
	t.Parallel()

	if assets.BaseCSS() == assets.HomeCSS() {
		t.Error("base.css and home.css are identical; the loader is reading the same file twice")
	}
}

// The CSS is loaded once in init() and handed out by reference, so the getters
// must keep returning the same content no matter how many callers ask.
func TestGettersAreStableAcrossCalls(t *testing.T) {
	t.Parallel()

	firstBase := assets.BaseCSS()
	firstHome := assets.HomeCSS()

	for i := 0; i < 3; i++ {
		if got := assets.BaseCSS(); got != firstBase {
			t.Fatalf("BaseCSS returned different content on call %d", i+2)
		}
		if got := assets.HomeCSS(); got != firstHome {
			t.Fatalf("HomeCSS returned different content on call %d", i+2)
		}
	}
}

func truncate(s string) string {
	const max = 120
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

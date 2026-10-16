package assets_test

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/assets"
)

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

func TestGettersAreStableAcrossCalls(t *testing.T) {
	t.Parallel()

	firstBase := assets.BaseCSS()
	firstHome := assets.HomeCSS()

	for i := range 3 {
		if got := assets.BaseCSS(); got != firstBase {
			t.Fatalf("BaseCSS returned different content on call %d", i+2)
		}
		if got := assets.HomeCSS(); got != firstHome {
			t.Fatalf("HomeCSS returned different content on call %d", i+2)
		}
	}
}

func truncate(s string) string {
	const limit = 120
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}

func TestErrorPagesRenderedWithoutARequestAreStandalone(t *testing.T) {
	t.Parallel()

	for _, page := range assets.ErrorPages {
		for _, tailwind := range []bool{false, true} {
			got := assets.ErrorTemplate(page, tailwind)
			// Django renders 400.html and 500.html with an empty Context and no request.
			if page.Code != "400" && page.Code != "500" {
				continue
			}
			if strings.Contains(got, "{%") || strings.Contains(got, "{{") {
				t.Errorf("%s.html (tailwind=%v) uses template tags; it must survive a broken "+
					"context processor, static manifest or database:\n%s", page.Code, tailwind, got)
			}
		}
	}
}

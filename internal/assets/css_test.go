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

func TestEveryLayoutShowsFlashMessages(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		for _, htmx := range []bool{false, true} {
			base := assets.RenderPage(assets.PageBase, assets.PageOptions{Tailwind: tailwind, HTMX: htmx, AllAuth: true, GlobalTemplates: true})
			// Django clears messages only when they are iterated, so a page that
			// skips them leaks every one onto whichever page iterates next.
			if !strings.Contains(base, "{% for message in messages %}") {
				t.Errorf("base.html (tailwind=%v, htmx=%v) never iterates messages:\n%s", tailwind, htmx, base)
			}
		}
	}
}

func TestNavigationStaysReachableOnPhones(t *testing.T) {
	t.Parallel()

	tailwind := assets.RenderPage(assets.PageBase, assets.PageOptions{Tailwind: true, GlobalTemplates: true})
	if strings.Contains(tailwind, `<nav class="hidden`) {
		t.Errorf("the Tailwind nav is hidden below md with no menu to replace it:\n%s", tailwind)
	}
	if strings.Contains(assets.BaseCSS(), "nav { display: none; }") {
		t.Error("base.css hides the nav on narrow screens with no menu to replace it")
	}
}

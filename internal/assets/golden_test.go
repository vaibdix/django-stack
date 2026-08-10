package assets_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/assets"
)

// Golden coverage for the generated HTML templates.
//
// These goldens were captured against the original stringly-typed
// TemplateContent(string, map[string]string) API. They are unchanged across the
// move to RenderPage(Page, PageOptions) — which is the proof that the typed
// replacement renders byte-identical output.
//
//	go test ./internal/assets -update

var updateGolden = flag.Bool("update", false, "rewrite golden files instead of comparing against them")

func assertGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", "golden", name)

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating golden dir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("writing golden %s: %v", name, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden %s: %v\n\nRun: go test ./internal/assets -update", name, err)
	}
	if got != string(want) {
		t.Errorf("rendered %s does not match its golden file.\n"+
			"If intentional, run: go test ./internal/assets -update", name)
	}
}

type featureSet struct {
	name string
	opts assets.PageOptions
}

func featureSets() []featureSet {
	base := func(apply func(*assets.PageOptions)) assets.PageOptions {
		opts := assets.PageOptions{AppName: "catalog", GlobalTemplates: true}
		if apply != nil {
			apply(&opts)
		}
		return opts
	}

	return []featureSet{
		{name: "plain", opts: base(nil)},
		{name: "tailwind", opts: base(func(o *assets.PageOptions) { o.Tailwind = true })},
		{name: "htmx", opts: base(func(o *assets.PageOptions) { o.HTMX = true })},
		{name: "api", opts: base(func(o *assets.PageOptions) { o.API = true })},
		{name: "allauth", opts: base(func(o *assets.PageOptions) { o.AllAuth = true })},
		{name: "tailwind_api_htmx_allauth", opts: base(func(o *assets.PageOptions) {
			o.Tailwind, o.API, o.HTMX, o.AllAuth = true, true, true, true
		})},
	}
}

func TestGoldenRenderPage(t *testing.T) {
	pages := []assets.Page{assets.PageBase, assets.PageIndex, assets.PageAPIDocs}

	for _, fs := range featureSets() {
		for _, page := range pages {
			name := fs.name + "/" + string(page)
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assertGolden(t, name, assets.RenderPage(page, fs.opts))
			})
		}
	}
}

func TestRenderPageUnknownPageReturnsEmpty(t *testing.T) {
	t.Parallel()

	got := assets.RenderPage(assets.Page("does-not-exist.html"), assets.PageOptions{})

	if strings.TrimSpace(got) != "" {
		t.Errorf("RenderPage for an unknown page returned %d bytes, want empty", len(got))
	}
}

func TestBaseTemplateWiresHTMXCompletely(t *testing.T) {
	t.Parallel()

	got := assets.RenderPage(assets.PageBase, assets.PageOptions{
		GlobalTemplates: true, HTMX: true, AppName: "catalog",
	})

	for _, needle := range []string{"{% load django_htmx %}", "{% htmx_script %}", "hx-headers"} {
		if !strings.Contains(got, needle) {
			t.Errorf("HTMX enabled but base.html is missing %q", needle)
		}
	}
}

func TestBaseTemplateOmitsHTMXWhenDisabled(t *testing.T) {
	t.Parallel()

	got := assets.RenderPage(assets.PageBase, assets.PageOptions{
		GlobalTemplates: true, AppName: "catalog",
	})

	for _, needle := range []string{"django_htmx", "htmx_script", "hx-headers"} {
		if strings.Contains(got, needle) {
			t.Errorf("HTMX disabled but base.html still contains %q", needle)
		}
	}
}

// The API docs link is driven by the "api" flag rather than by the presence of
// a REST app, so it is worth pinning both directions.
func TestBaseTemplateAPILinkFollowsFlag(t *testing.T) {
	t.Parallel()

	enabled := assets.RenderPage(assets.PageBase, assets.PageOptions{
		GlobalTemplates: true, API: true, AppName: "catalog",
	})
	disabled := assets.RenderPage(assets.PageBase, assets.PageOptions{
		GlobalTemplates: true, AppName: "catalog",
	})

	if !strings.Contains(enabled, "swagger-ui") {
		t.Error("api=true but base.html has no swagger-ui link")
	}
	if strings.Contains(disabled, "swagger-ui") {
		t.Error("api=false but base.html still links to swagger-ui")
	}
}

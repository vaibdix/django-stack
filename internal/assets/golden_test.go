package assets_test

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/golden"

	"github.com/vaibdix/django-cli/internal/assets"
)

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
	pages := []assets.Page{assets.PageBase, assets.PageIndex}

	for _, fs := range featureSets() {
		for _, page := range pages {
			name := fs.name + "/" + string(page)
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				golden.Assert(t, name, assets.RenderPage(page, fs.opts))
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

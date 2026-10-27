package render

import (
	"strings"
	"testing"
)

func TestHtmxViewsPickTheirTemplateThroughTheTypedMixin(t *testing.T) {
	t.Parallel()

	views := renderBookViews(bookViewsData{AppName: "catalog", HTMX: true, RuntimeGenerics: true})

	for _, want := range []string{
		"from .htmx import HtmxTemplateMixin",
		"HtmxTemplateMixin,",
	} {
		if !strings.Contains(views, want) {
			t.Errorf("htmx views are missing %q, so request.htmx would be read off an untyped request:\n%s", want, views)
		}
	}
}

func TestNonHtmxViewsReferenceNoHtmxModule(t *testing.T) {
	t.Parallel()

	views := renderBookViews(bookViewsData{AppName: "catalog", HTMX: false, RuntimeGenerics: true})

	if strings.Contains(views, "htmx") {
		t.Errorf("non-htmx views reference a module their project does not generate:\n%s", views)
	}
}

func TestHtmxSupportDeclaresTheRequestTypeRatherThanImportingIt(t *testing.T) {
	t.Parallel()

	support := generateHtmxSupport()

	if strings.Contains(support, "import HtmxHttpRequest") {
		t.Errorf("htmx.py imports a name django-htmx does not export:\n%s", support)
	}
	for _, want := range []string{
		"from django_htmx.middleware import HtmxDetails",
		"class HtmxHttpRequest(HttpRequestBase):",
		"htmx: HtmxDetails",
		"def wants_fragment(request: HttpRequestBase) -> bool:",
		"class HtmxTemplateMixin(TemplateResponseMixin):",
		"def get_template_names(self) -> list[str]:",
	} {
		if !strings.Contains(support, want) {
			t.Errorf("htmx.py is missing %q:\n%s", want, support)
		}
	}
}

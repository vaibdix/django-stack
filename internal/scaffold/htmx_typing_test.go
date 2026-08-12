package scaffold

import (
	"strings"
	"testing"
)

func TestHtmxViewsUseTheRequestTypeThatCarriesHtmx(t *testing.T) {
	t.Parallel()

	views := generateBookViews("catalog", true, false)

	if strings.Contains(views, "request: HttpRequest") {
		t.Errorf("htmx views annotate the request as plain HttpRequest, so request.htmx does not type-check:\n%s", views)
	}
	if !strings.Contains(views, "request: HtmxHttpRequest") {
		t.Errorf("htmx views do not use HtmxHttpRequest:\n%s", views)
	}
	if !strings.Contains(views, "from .htmx import HtmxHttpRequest") {
		t.Errorf("htmx views do not import the request type:\n%s", views)
	}
}

func TestNonHtmxViewsKeepThePlainRequestType(t *testing.T) {
	t.Parallel()

	views := generateBookViews("catalog", false, false)

	if !strings.Contains(views, "request: HttpRequest") {
		t.Errorf("non-htmx views should annotate with HttpRequest:\n%s", views)
	}
	if strings.Contains(views, "HtmxHttpRequest") {
		t.Errorf("non-htmx views reference a type their project does not generate:\n%s", views)
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
		"def wants_fragment(request: HtmxHttpRequest) -> bool:",
	} {
		if !strings.Contains(support, want) {
			t.Errorf("htmx.py is missing %q:\n%s", want, support)
		}
	}
}

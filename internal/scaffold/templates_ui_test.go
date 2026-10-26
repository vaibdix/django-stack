package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

func TestMainJSIsAClassicScript(t *testing.T) {
	t.Parallel()

	for _, stack := range []config.Stack{config.StackStandard, config.StackHTMX} {
		js := (&Env{cfg: mustSpec(config.Choices{Stack: stack})}).mainJS()
		for line := range strings.SplitSeq(js, "\n") {
			if strings.HasPrefix(line, "export") || strings.HasPrefix(line, "import ") {
				t.Errorf("%s: base.html loads main.js without type=module, so %q is a SyntaxError", stack, line)
			}
		}
	}
}

func TestTailwindTemplateContentMatchesTheReferenceLayout(t *testing.T) {
	t.Parallel()

	base := assets.RenderPage(assets.PageBase, assets.PageOptions{
		Tailwind: true,
		AppName:  "library",
		API:      true,
	})
	if !strings.Contains(base, "{% url 'swagger-ui' %}") {
		t.Fatalf("expected tailwind base template to include API docs nav link, got:\n%s", base)
	}
	if !strings.Contains(base, `class="frame"`) {
		t.Fatalf("expected tailwind base template to wrap the page in the framed column, got:\n%s", base)
	}
	if !strings.Contains(base, `id="next-steps"`) {
		t.Fatalf("expected the reference's closing band, got:\n%s", base)
	}
	if strings.Contains(base, `id="spacer"`) {
		t.Fatalf("the reference's spacer closes its column, but this layout has a footer doing that; "+
			"keeping both is 88px of dead height, got:\n%s", base)
	}
	if strings.Contains(base, "white/") {
		t.Fatalf("expected the base template to style with tokens rather than opacity literals, got:\n%s", base)
	}
	if strings.Contains(base, `class="dark"`) {
		t.Fatalf("the layout pins a colour scheme; the reference follows prefers-color-scheme, got:\n%s", base)
	}

	index := assets.RenderPage(assets.PageIndex, assets.PageOptions{Tailwind: true})
	if !strings.Contains(index, "Get started") {
		t.Fatalf("expected tailwind index template to include the reference heading, got:\n%s", index)
	}
	if !strings.Contains(index, "Open admin") {
		t.Fatalf("expected tailwind index template to include admin CTA, got:\n%s", index)
	}
	if !strings.Contains(index, "button button-primary") || !strings.Contains(index, "button button-secondary") {
		t.Fatalf("expected the index CTAs to use the shared button classes, got:\n%s", index)
	}
}

func TestBuildProjectURLsContentMatchesEnabledFeatures(t *testing.T) {
	t.Parallel()

	minimal := (&Env{}).urlsText()
	if strings.Contains(minimal, "HomeView") || strings.Contains(minimal, "SpectacularAPIView") {
		t.Fatalf("minimal URL config should not reference template views, got:\n%s", minimal)
	}

	apiOnly := (&Env{cfg: mustSpec(config.Choices{
		App:    "library",
		AddOns: config.NewSet(config.RestFramework),
	})}).urlsText()
	if !strings.Contains(apiOnly, "api-auth/") || !strings.Contains(apiOnly, "api/v1/") {
		t.Fatalf("API URL config should include REST routes, got:\n%s", apiOnly)
	}
	if !strings.Contains(apiOnly, "SpectacularAPIView") {
		t.Fatalf("API URL config should include drf-spectacular routes, got:\n%s", apiOnly)
	}

	full := (&Env{cfg: mustSpec(config.Choices{
		App:    "library",
		Stack:  config.StackStandard,
		AddOns: config.NewSet(config.RestFramework),
	})}).urlsText()
	required := []string{"views.HomeView", "SpectacularSwaggerSplitView", "api-auth/", "api/v1/", "library/"}
	for _, fragment := range required {
		if !strings.Contains(full, fragment) {
			t.Fatalf("full URL config missing %q, got:\n%s", fragment, full)
		}
	}
}

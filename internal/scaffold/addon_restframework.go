package scaffold

import (
	"context"
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
)

func restFrameworkAddOn() addOn {
	return addOn{
		enabled: has(config.RestFramework),
		steps:   []phasedStep{{phase: phaseAPI, step: step{name: "setup rest framework", label: "Setting up Django REST Framework...", run: (*Env).applyRESTFramework}}},
		contribute: func(p project) contribution {
			return contribution{
				runtime: []string{pkgRestFramework, pkgDRFSpectacular, pkgDRFSpectacularSidecar},
				readme:  func(d *readmeData) { d.REST, d.DRF = true, true },
				urls:    p.restFrameworkURLs(),
				settings: settingsContribution{
					lateApps:   []string{appRestFramework, appDRFSpectacular, appDRFSpectacularSidecar},
					lateBlocks: []string{restFrameworkSettings(), p.spectacularSettings()},
				},
			}
		},
	}
}

func (p project) restFrameworkURLs() urlContribution {
	nativeCSP := config.DjangoSupportsNativeCSP(p.cfg.Django)
	urls := urlContribution{
		home:     "/api/schema/swagger-ui/",
		imports:  []string{"from drf_spectacular.views import SpectacularAPIView, SpectacularSwaggerSplitView"},
		blocks:   []string{swaggerViewWithInlineStyles(nativeCSP)},
		appRoute: true,
		patterns: []string{
			`    path("api/schema/", SpectacularAPIView.as_view(), name="schema"),`,
			`    path("api/schema/swagger-ui/", swagger_ui, name="swagger-ui"),`,
			`    path("api-auth/", include("rest_framework.urls", namespace="rest_framework")),`,
		},
	}
	if nativeCSP {
		urls.imports = append(urls.imports, "from django.utils.csp import CSP", "from django.views.decorators.csp import csp_override, csp_report_only_override")
	} else {
		urls.imports = append(urls.imports, "from csp.constants import UNSAFE_INLINE", "from csp.decorators import csp_update")
	}
	if p.cfg.App != "" {
		urls.patterns = append(urls.patterns, fmt.Sprintf(`    path("api/v1/", include("%s.api")),`, p.cfg.App))
		if !p.cfg.APIOnly() {
			urls.patterns = append(urls.patterns, fmt.Sprintf(`    path("%s/", include("%s.urls")),`, p.cfg.App, p.cfg.App))
		}
	}
	return urls
}

func (e *Env) applyRESTFramework(ctx context.Context) error {
	if e.cfg.App != "" {
		if err := e.writeAPIPackage(); err != nil {
			return err
		}
	}

	return nil
}

func swaggerViewWithInlineStyles(nativeCSP bool) string {
	if nativeCSP {
		return `swagger_ui = SpectacularSwaggerSplitView.as_view(url_name="schema")
if settings.SECURE_CSP:
    swagger_ui = csp_override({**settings.SECURE_CSP, "style-src": [CSP.SELF, CSP.UNSAFE_INLINE]})(swagger_ui)
if settings.SECURE_CSP_REPORT_ONLY:
    swagger_ui = csp_report_only_override(
        {**settings.SECURE_CSP_REPORT_ONLY, "style-src": [CSP.SELF, CSP.UNSAFE_INLINE]},
    )(swagger_ui)`
	}
	return `swagger_ui = csp_update({"style-src": [UNSAFE_INLINE]}, REPORT_ONLY=True)(
    csp_update({"style-src": [UNSAFE_INLINE]})(SpectacularSwaggerSplitView.as_view(url_name="schema")),
)`
}

func restFrameworkSettings() string { return renderStatic("rest_framework.py.tmpl") }

func (p project) spectacularSettings() string {
	return render("spectacular.py.tmpl", struct{ Name string }{p.cfg.Name})
}

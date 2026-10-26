package scaffold

import (
	"context"
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
)

func restFrameworkFeature() feature {
	return feature{
		enabled: has(config.RestFramework),
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
	nativeCSP := p.cfg.Django().SupportsNativeCSP()
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
	if p.cfg.App().Ok() {
		urls.patterns = append(urls.patterns, fmt.Sprintf(`    path("api/v1/", include("%s.api")),`, p.cfg.App().String()))
		if !p.cfg.APIOnly() {
			urls.patterns = append(urls.patterns, fmt.Sprintf(`    path("%s/", include("%s.urls")),`, p.cfg.App().String(), p.cfg.App().String()))
		}
	}
	return urls
}

func (e *Env) applyRESTFramework(ctx context.Context) error {
	if e.cfg.App().Ok() {
		if err := e.writeAPIPackage(); err != nil {
			return err
		}
	}

	return nil
}

func swaggerViewWithInlineStyles(nativeCSP bool) string {
	return render("swagger_view.py.tmpl", struct{ NativeCSP bool }{nativeCSP})
}

func restFrameworkSettings() string { return renderStatic("rest_framework.py.tmpl") }

func (p project) spectacularSettings() string {
	return render("spectacular.py.tmpl", struct{ Name string }{p.cfg.Name().String()})
}

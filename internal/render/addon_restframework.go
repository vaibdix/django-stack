package render

import (
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
)

func restFrameworkFeature() feature {
	return feature{
		enabled: Has(config.RestFramework),
		contribute: func(p Project) contribution {
			return contribution{
				runtime: []string{PkgRestFramework, pkgDRFSpectacular, pkgDRFSpectacularSidecar},
				readme:  func(d *readmeData) { d.REST, d.DRF = true, true },
				urls:    p.restFrameworkURLs(),
				settings: settingsContribution{
					apps: []entry{
						{name: appRestFramework, after: atEnd, rank: rankAPI},
						{name: appDRFSpectacular, after: appRestFramework},
						{name: appDRFSpectacularSidecar, after: appDRFSpectacular},
					},
					blocks: []block{
						{section: afterUserModel, rank: rankAPI, text: restFrameworkSettings()},
						{section: afterUserModel, rank: rankAPI + 1, text: p.spectacularSettings()},
					},
				},
			}
		},
	}
}

func (p Project) restFrameworkURLs() urlContribution {
	nativeCSP := p.Cfg.Django().SupportsNativeCSP()
	urls := urlContribution{
		home:     "/api/schema/swagger-ui/",
		imports:  []string{"from drf_spectacular.views import SpectacularAPIView, SpectacularSwaggerSplitView"},
		blocks:   []block{{rank: rankAPI, text: swaggerViewWithInlineStyles(nativeCSP)}},
		appRoute: true,
		routes: []route{
			{rank: rankAPIRoute, line: `    path("api/schema/", SpectacularAPIView.as_view(), name="schema"),`},
			{rank: rankAPIRoute + 1, line: `    path("api/schema/swagger-ui/", swagger_ui, name="swagger-ui"),`},
			{rank: rankAPIRoute + 2, line: `    path("api-auth/", include("rest_framework.urls", namespace="rest_framework")),`},
		},
	}
	if nativeCSP {
		urls.imports = append(urls.imports, "from django.utils.csp import CSP", "from django.views.decorators.csp import csp_override, csp_report_only_override")
	} else {
		urls.imports = append(urls.imports, "from csp.constants import UNSAFE_INLINE", "from csp.decorators import csp_update")
	}
	if p.Cfg.App().Ok() {
		urls.routes = append(urls.routes, route{rank: rankAPIRoute + 3, line: fmt.Sprintf(`    path("api/v1/", include("%s.api")),`, p.Cfg.App().String())})
		if !p.Cfg.APIOnly() {
			urls.routes = append(urls.routes, route{rank: rankAPIRoute + 4, line: fmt.Sprintf(`    path("%s/", include("%s.urls")),`, p.Cfg.App().String(), p.Cfg.App().String())})
		}
	}
	return urls
}

func swaggerViewWithInlineStyles(nativeCSP bool) string {
	return Render("swagger_view.py.tmpl", struct{ NativeCSP bool }{nativeCSP})
}

func restFrameworkSettings() string { return RenderStatic("rest_framework.py.tmpl") }

func (p Project) spectacularSettings() string {
	return Render("spectacular.py.tmpl", struct{ Name string }{p.Cfg.Name().String()})
}

from django.conf import settings
from django.contrib import admin
from django.urls import include
from django.urls import path
from django.utils.csp import CSP
from django.views.decorators.csp import csp_override
from django.views.decorators.csp import csp_report_only_override
from drf_spectacular.views import SpectacularAPIView
from drf_spectacular.views import SpectacularSwaggerSplitView

from . import views

swagger_ui = SpectacularSwaggerSplitView.as_view(url_name="schema")
if settings.SECURE_CSP:
    swagger_ui = csp_override({**settings.SECURE_CSP, "style-src": [CSP.SELF, CSP.UNSAFE_INLINE]})(swagger_ui)
if settings.SECURE_CSP_REPORT_ONLY:
    swagger_ui = csp_report_only_override(
        {**settings.SECURE_CSP_REPORT_ONLY, "style-src": [CSP.SELF, CSP.UNSAFE_INLINE]},
    )(swagger_ui)

urlpatterns = [
    path("", views.HomeView.as_view(), name="home"),
    path(settings.ADMIN_URL, admin.site.urls),
    path("healthz/", views.healthz, name="healthz"),
    path("accounts/", include("allauth.urls")),
    path("api/schema/", SpectacularAPIView.as_view(), name="schema"),
    path("api/schema/swagger-ui/", swagger_ui, name="swagger-ui"),
    path("api-auth/", include("rest_framework.urls", namespace="rest_framework")),
    path("api/v1/", include("catalog.api")),
    path("catalog/", include("catalog.urls")),
]

if settings.DEBUG:
    urlpatterns += [path("__reload__/", include("django_browser_reload.urls"))]

from django.core.exceptions import ImproperlyConfigured

from .base import *  # noqa: F401,F403

if ENVIRONMENT != "local":
    message = (
        f"settings.local refuses to run with ENVIRONMENT={ENVIRONMENT}. manage.py picks settings.local "
        "unless DJANGO_SETTINGS_MODULE is exported, and .env is read too late to change that: "
        "export DJANGO_SETTINGS_MODULE=my_shop.settings.production on the server."
    )
    raise ImproperlyConfigured(message)

ALLOWED_HOSTS = ["localhost", "127.0.0.1"]

INSTALLED_APPS = [*INSTALLED_APPS, "django_browser_reload"]

MIDDLEWARE = [*MIDDLEWARE, "django_browser_reload.middleware.BrowserReloadMiddleware"]

EMAIL_BACKEND = "django.core.mail.backends.console.EmailBackend"

from .base import *  # noqa: F401,F403

ALLOWED_HOSTS = ["localhost", "127.0.0.1"]

INSTALLED_APPS = [*INSTALLED_APPS, "django_browser_reload"]

MIDDLEWARE = [*MIDDLEWARE, "django_browser_reload.middleware.BrowserReloadMiddleware"]

MAILERS = {
    "default": {
        "BACKEND": "django.core.mail.backends.console.EmailBackend",
    },
}


from typing import Any

from .base import *  # noqa: F401,F403

DEBUG = False

SECRET_KEY = env.SECRET_KEY

ALLOWED_HOSTS = env.ALLOWED_HOSTS

SECURE_SSL_REDIRECT = env.SECURE_SSL_REDIRECT
SECURE_HSTS_SECONDS = env.SECURE_HSTS_SECONDS
SECURE_HSTS_INCLUDE_SUBDOMAINS = True
SECURE_HSTS_PRELOAD = True
SECURE_PROXY_SSL_HEADER = ("HTTP_X_FORWARDED_PROTO", "https")
SESSION_COOKIE_SECURE = True
CSRF_COOKIE_SECURE = True

ACCOUNT_EMAIL_VERIFICATION = "mandatory"


# The mailer is annotated and named privately for two reasons: it mixes a string
# with a nested mapping, which mypy would otherwise narrow to dict[str, str]
# from the first entry, and annotating MAILERS itself would read as a
# redefinition of a name the star import above may already have bound.
_DEFAULT_MAILER: dict[str, Any] = {
    "BACKEND": "django.core.mail.backends.smtp.EmailBackend",
    "OPTIONS": {
        "host": env.EMAIL_HOST,
        "port": env.EMAIL_PORT,
        "username": env.EMAIL_HOST_USER,
        "password": env.EMAIL_HOST_PASSWORD,
        "use_tls": env.EMAIL_USE_TLS,
    },
}

MAILERS = {"default": _DEFAULT_MAILER}

STORAGES = {
    "default": {
        "BACKEND": "django.core.files.storage.FileSystemStorage",
    },
    "staticfiles": {
        "BACKEND": "django.contrib.staticfiles.storage.ManifestStaticFilesStorage",
    },
}

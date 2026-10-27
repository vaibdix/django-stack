from typing import TypedDict

from django.core.exceptions import ImproperlyConfigured

from . import base
from .base import *  # noqa: F401,F403

if ENVIRONMENT == "local":
    message = "settings.production refuses to run with ENVIRONMENT=local; set it to staging or production."
    raise ImproperlyConfigured(message)

if SECRET_KEY == "replace-this-placeholder-with-a-generated-50-char-key":  # noqa: S105
    message = "SECRET_KEY is still the .env.example placeholder; generate a real one before deploying."
    raise ImproperlyConfigured(message)

ALLOWED_HOSTS = env.ALLOWED_HOSTS

CSRF_TRUSTED_ORIGINS = env.CSRF_TRUSTED_ORIGINS

SECURE_CSP = base.SECURE_CSP_REPORT_ONLY
SECURE_CSP_REPORT_ONLY = {}

SECURE_SSL_REDIRECT = env.SECURE_SSL_REDIRECT
SECURE_REDIRECT_EXEMPT = [r"^healthz/$"]
SECURE_HSTS_SECONDS = env.SECURE_HSTS_SECONDS
SECURE_HSTS_INCLUDE_SUBDOMAINS = env.SECURE_HSTS_INCLUDE_SUBDOMAINS
SECURE_HSTS_PRELOAD = env.SECURE_HSTS_PRELOAD
SESSION_COOKIE_SECURE = True
CSRF_COOKIE_SECURE = True

if env.TRUST_PROXY_SSL_HEADER:
    SECURE_PROXY_SSL_HEADER = ("HTTP_X_FORWARDED_PROTO", "https")

DEFAULT_FROM_EMAIL = env.DEFAULT_FROM_EMAIL
SERVER_EMAIL = DEFAULT_FROM_EMAIL

if DEFAULT_FROM_EMAIL == "webmaster@localhost":
    message = "DEFAULT_FROM_EMAIL is still webmaster@localhost; set it to an address your mail server sends from."
    raise ImproperlyConfigured(message)


class MailerOptions(TypedDict):
    host: str
    port: int
    username: str
    password: str
    use_tls: bool


class MailerConfig(TypedDict):
    BACKEND: str
    OPTIONS: MailerOptions


_DEFAULT_MAILER: MailerConfig = {
    "BACKEND": "django.core.mail.backends.smtp.EmailBackend",
    "OPTIONS": {
        "host": env.EMAIL_HOST,
        "port": env.EMAIL_PORT,
        "username": env.EMAIL_HOST_USER,
        "password": env.EMAIL_HOST_PASSWORD.get_secret_value(),
        "use_tls": env.EMAIL_USE_TLS,
    },
}

MAILERS = {"default": _DEFAULT_MAILER}

CACHES = {
    "default": {
        "BACKEND": "django.core.cache.backends.db.DatabaseCache",
        "LOCATION": "django_cache_table",
    },
}

STORAGES = {
    "default": {
        "BACKEND": "django.core.files.storage.FileSystemStorage",
    },
    "staticfiles": {
        "BACKEND": "whitenoise.storage.CompressedManifestStaticFilesStorage",
    },
}

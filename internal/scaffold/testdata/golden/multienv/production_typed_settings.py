from typing import TypedDict

from django.core.exceptions import ImproperlyConfigured

from .base import *  # noqa: F401,F403

if ENVIRONMENT == "local":
    message = "settings.production refuses to run with ENVIRONMENT=local; set it to staging or production."
    raise ImproperlyConfigured(message)

SECRET_KEY = env.SECRET_KEY.get_secret_value()

ALLOWED_HOSTS = env.ALLOWED_HOSTS

CSRF_TRUSTED_ORIGINS = env.CSRF_TRUSTED_ORIGINS

SECURE_SSL_REDIRECT = env.SECURE_SSL_REDIRECT
SECURE_HSTS_SECONDS = env.SECURE_HSTS_SECONDS
SECURE_HSTS_INCLUDE_SUBDOMAINS = True
SECURE_HSTS_PRELOAD = True
SESSION_COOKIE_SECURE = True
SESSION_COOKIE_SAMESITE = "Lax"
CSRF_COOKIE_SECURE = True
CSRF_COOKIE_SAMESITE = "Lax"

if env.TRUST_PROXY_SSL_HEADER:
    SECURE_PROXY_SSL_HEADER = ("HTTP_X_FORWARDED_PROTO", "https")

ACCOUNT_EMAIL_VERIFICATION = "mandatory"

DEFAULT_FROM_EMAIL = env.DEFAULT_FROM_EMAIL
SERVER_EMAIL = DEFAULT_FROM_EMAIL


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

CONN_MAX_AGE = 60

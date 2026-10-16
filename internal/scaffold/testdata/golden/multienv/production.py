from typing import TypedDict

from django.core.exceptions import ImproperlyConfigured

from .base import *  # noqa: F401,F403

if ENVIRONMENT == "local":
    message = "settings.production refuses to run with ENVIRONMENT=local; set it to staging or production."
    raise ImproperlyConfigured(message)

SECRET_KEY = env('SECRET_KEY')

ALLOWED_HOSTS = env.list('ALLOWED_HOSTS')

CSRF_TRUSTED_ORIGINS = env.list('CSRF_TRUSTED_ORIGINS', default=[])

SECURE_SSL_REDIRECT = env.bool('SECURE_SSL_REDIRECT', default=True)
SECURE_REDIRECT_EXEMPT = [r"^healthz/$"]
SECURE_HSTS_SECONDS = env.int('SECURE_HSTS_SECONDS', default=31536000)
SECURE_HSTS_INCLUDE_SUBDOMAINS = env.bool('SECURE_HSTS_INCLUDE_SUBDOMAINS', default=False)
SECURE_HSTS_PRELOAD = env.bool('SECURE_HSTS_PRELOAD', default=False)
SESSION_COOKIE_SECURE = True
SESSION_COOKIE_SAMESITE = "Lax"
CSRF_COOKIE_SECURE = True
CSRF_COOKIE_SAMESITE = "Lax"

if env.bool('TRUST_PROXY_SSL_HEADER', default=False):
    SECURE_PROXY_SSL_HEADER = ("HTTP_X_FORWARDED_PROTO", "https")

ACCOUNT_EMAIL_VERIFICATION = "mandatory"

DEFAULT_FROM_EMAIL = env('DEFAULT_FROM_EMAIL', default='webmaster@localhost')
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
        "host": env('EMAIL_HOST', default='localhost'),
        "port": env.int('EMAIL_PORT', default=587),
        "username": env('EMAIL_HOST_USER', default=''),
        "password": env('EMAIL_HOST_PASSWORD', default=''),
        "use_tls": env.bool('EMAIL_USE_TLS', default=True),
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

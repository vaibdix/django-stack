from typing import TypedDict

from .base import *  # noqa: F401,F403

DEBUG = False

SECRET_KEY = env('SECRET_KEY')

ALLOWED_HOSTS = env.list('ALLOWED_HOSTS')

SECURE_SSL_REDIRECT = env.bool('SECURE_SSL_REDIRECT', default=True)
SECURE_HSTS_SECONDS = env.int('SECURE_HSTS_SECONDS', default=31536000)
SECURE_HSTS_INCLUDE_SUBDOMAINS = True
SECURE_HSTS_PRELOAD = True
SECURE_PROXY_SSL_HEADER = ("HTTP_X_FORWARDED_PROTO", "https")
SESSION_COOKIE_SECURE = True
CSRF_COOKIE_SECURE = True

ACCOUNT_EMAIL_VERIFICATION = "mandatory"



class MailerOptions(TypedDict):
    host: str
    port: int
    username: str
    password: str
    use_tls: bool


class MailerConfig(TypedDict):
    BACKEND: str
    OPTIONS: MailerOptions


# Named privately because annotating MAILERS itself would read as a redefinition
# of a name the star import above may already have bound.
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

STORAGES = {
    "default": {
        "BACKEND": "django.core.files.storage.FileSystemStorage",
    },
    "staticfiles": {
        "BACKEND": "django.contrib.staticfiles.storage.ManifestStaticFilesStorage",
    },
}

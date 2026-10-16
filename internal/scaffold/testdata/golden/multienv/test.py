import os

os.environ.setdefault("ENVIRONMENT", "test")

from .base import *  # noqa: E402,F401,F403

PASSWORD_HASHERS = ["django.contrib.auth.hashers.MD5PasswordHasher"]

LOGGING = {"version": 1, "disable_existing_loggers": True}

MAILERS = {
    "default": {
        "BACKEND": "django.core.mail.backends.locmem.EmailBackend",
    },
}


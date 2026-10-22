import json
import os
import sys

sys.path.insert(0, os.getcwd())

import django

django.setup()

from django.conf import settings
from django.test import Client
from django.test.utils import setup_test_environment

setup_test_environment()
client = Client()
health = client.get("/healthz/")
secure = client.get("/healthz/", secure=True)
missing = client.get("/no-such-page-matrix/", secure=True)
print(
    json.dumps(
        {
            "debug": settings.DEBUG,
            "healthz": health.status_code,
            "csp": bool(secure.headers.get("Content-Security-Policy")),
            "notFound": missing.status_code,
        }
    )
)

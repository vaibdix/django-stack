from typing import TypedDict

from django.http import HttpRequest


class ProjectContext(TypedDict):
    project_name: str


def project_context(request: HttpRequest) -> ProjectContext:
    return {
        "project_name": "my-shop",
    }

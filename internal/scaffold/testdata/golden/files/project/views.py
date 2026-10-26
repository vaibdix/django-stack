from django.db import Error
from django.db import connection
from django.http import HttpRequest
from django.http import JsonResponse
from django.views.generic import TemplateView


class HomeView(TemplateView):
    template_name = "index.html"


def healthz(request: HttpRequest) -> JsonResponse:
    try:
        connection.ensure_connection()
    except Error:
        return JsonResponse({"status": "error", "database": "unreachable"}, status=503)
    return JsonResponse({"status": "ok", "database": "ok"})

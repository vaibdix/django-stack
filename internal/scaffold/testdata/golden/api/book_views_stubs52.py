from http import HTTPStatus

from django.contrib import messages
from django.contrib.auth.mixins import LoginRequiredMixin
from django.contrib.messages.views import SuccessMessageMixin
from django.forms import Form
from django.http import HttpRequest
from django.http import HttpResponse
from django.urls import reverse_lazy
from django.views.generic import CreateView
from django.views.generic import DeleteView
from django.views.generic import ListView
from django.views.generic import UpdateView
from django.views.generic.edit import ModelFormMixin

from .forms import BookForm
from .htmx import HtmxTemplateMixin
from .htmx import wants_fragment
from .models import Book
from .models import BookQuerySet

SEARCH_PARAM = "q"

PAGE_SIZE = 12

LIST_URL = reverse_lazy("catalog:book_list")

LIST_TEMPLATE = "catalog/book_list.html"
FORM_TEMPLATE = "catalog/book_form.html"
DELETE_TEMPLATE = "catalog/book_delete.html"
LIST_ITEMS_TEMPLATE = "catalog/partials/book_list_items.html"
FORM_PARTIAL_TEMPLATE = "catalog/partials/book_form_partial.html"


class BookListView(HtmxTemplateMixin, ListView[Book]):
    context_object_name = "books"
    paginate_by = PAGE_SIZE
    template_name = LIST_TEMPLATE
    fragment_template_name = LIST_ITEMS_TEMPLATE

    def setup(self, request: HttpRequest, *args: object, **kwargs: object) -> None:
        super().setup(request, *args, **kwargs)
        self.extra_context = {"query": self.search_term}

    def get_queryset(self) -> BookQuerySet:
        return Book.objects.search(self.search_term)

    @property
    def search_term(self) -> str:
        return self.request.GET.get(SEARCH_PARAM, "")


class BookFormMixin(
    LoginRequiredMixin,
    HtmxTemplateMixin,
    SuccessMessageMixin[BookForm],
    ModelFormMixin[Book, BookForm],
):
    model = Book
    form_class = BookForm
    success_url = LIST_URL
    template_name: str | None = FORM_TEMPLATE
    fragment_template_name = FORM_PARTIAL_TEMPLATE

    def form_valid(self, form: BookForm) -> HttpResponse:
        response = super().form_valid(form)
        if wants_fragment(self.request):
            return hand_back_to_the_browser(response["Location"])
        return response

    def form_invalid(self, form: BookForm) -> HttpResponse:
        response = super().form_invalid(form)
        if wants_fragment(self.request):
            response.status_code = HTTPStatus.UNPROCESSABLE_ENTITY
        return response


class BookCreateView(BookFormMixin, CreateView[Book, BookForm]):
    extra_context = {"action": "Create"}
    success_message = "Book created successfully!"


class BookUpdateView(BookFormMixin, UpdateView[Book, BookForm]):
    extra_context = {"action": "Edit"}
    success_message = "Book updated successfully!"


class BookDeleteView(LoginRequiredMixin, DeleteView[Book, Form]):  # type: ignore[type-var]
    model = Book
    success_url = LIST_URL
    template_name = DELETE_TEMPLATE

    def form_valid(self, form: Form) -> HttpResponse:
        deleted = super().form_valid(form)
        if wants_fragment(self.request):
            return HttpResponse("")
        messages.success(self.request, "Book deleted successfully!")
        return deleted


def hand_back_to_the_browser(url: str) -> HttpResponse:
    response = HttpResponse(status=HTTPStatus.NO_CONTENT)
    response["HX-Redirect"] = url
    return response



from django.contrib import messages
from django.contrib.auth.mixins import LoginRequiredMixin
from django.forms import Form
from django.http import HttpRequest
from django.http import HttpResponse
from django.urls import reverse_lazy
from django.views.generic import CreateView
from django.views.generic import DeleteView
from django.views.generic import ListView
from django.views.generic import UpdateView

from .forms import BookForm
from .models import Book
from .models import BookQuerySet

SEARCH_PARAM = 'q'

PAGE_SIZE = 12

LIST_URL = reverse_lazy('catalog:book_list')

LIST_TEMPLATE = 'catalog/book_list.html'
FORM_TEMPLATE = 'catalog/book_form.html'
DELETE_TEMPLATE = 'catalog/book_delete.html'


class BookListView(ListView[Book]):
    context_object_name = 'books'
    paginate_by = PAGE_SIZE
    template_name = LIST_TEMPLATE

    def setup(self, request: HttpRequest, *args: object, **kwargs: object) -> None:
        super().setup(request, *args, **kwargs)
        self.extra_context = {'query': self.search_term}

    def get_queryset(self) -> BookQuerySet:
        return Book.objects.search(self.search_term)

    @property
    def search_term(self) -> str:
        return self.request.GET.get(SEARCH_PARAM, '')


class BookCreateView(
    LoginRequiredMixin,
    CreateView[Book, BookForm],
):
    model = Book
    form_class = BookForm
    success_url = LIST_URL
    template_name = FORM_TEMPLATE

    def get_context_data(self, **kwargs: object) -> dict[str, object]:
        context = super().get_context_data(**kwargs)
        context['action'] = 'Create'
        return context

    def form_valid(self, form: BookForm) -> HttpResponse:
        messages.success(self.request, 'Book created successfully!')
        return super().form_valid(form)


class BookUpdateView(
    LoginRequiredMixin,
    UpdateView[Book, BookForm],
):
    model = Book
    form_class = BookForm
    success_url = LIST_URL
    template_name = FORM_TEMPLATE

    def get_context_data(self, **kwargs: object) -> dict[str, object]:
        context = super().get_context_data(**kwargs)
        context['action'] = 'Edit'
        return context

    def form_valid(self, form: BookForm) -> HttpResponse:
        messages.success(self.request, 'Book updated successfully!')
        return super().form_valid(form)


class BookDeleteView(LoginRequiredMixin, DeleteView[Book, Form]):
    model = Book
    success_url = LIST_URL
    template_name = DELETE_TEMPLATE

    def form_valid(self, form: Form) -> HttpResponse:
        messages.success(self.request, 'Book deleted successfully!')
        return super().form_valid(form)

from django.contrib import messages
from django.http import HttpRequest
from django.http import HttpResponse
from django.shortcuts import get_object_or_404
from django.shortcuts import redirect
from django.shortcuts import render
from django.urls import reverse

from .forms import BookForm
from .models import Book

app_name_var = 'catalog'


def wants_fragment(request: HttpRequest) -> bool:
    """True for a real partial request, False for boosted navigation.

    hx-boost sets HX-Request on ordinary link and form navigation, which still
    needs the whole page. Only a targeted hx-get/hx-post should get a fragment.
    """
    return bool(request.htmx) and not request.htmx.boosted


def book_list(request: HttpRequest) -> HttpResponse:
    books = Book.objects.all()
    if wants_fragment(request):
        return render(request, app_name_var + '/partials/book_list_items.html', {'books': books})
    return render(request, app_name_var + '/book_list.html', {'books': books})


def book_create(request: HttpRequest) -> HttpResponse:
    if request.method == 'POST':
        form = BookForm(request.POST)
        if form.is_valid():
            form.save()
            messages.success(request, 'Book created successfully!')
            if wants_fragment(request):
                return render(request, app_name_var + '/partials/book_list_items.html', {'books': Book.objects.all()})
            return redirect(reverse(app_name_var + ':book_list'))
    else:
        form = BookForm()
    if wants_fragment(request):
        template = app_name_var + '/partials/book_form_partial.html'
    else:
        template = app_name_var + '/book_form.html'
    return render(request, template, {'form': form, 'action': 'Create'})


def book_edit(request: HttpRequest, pk: int) -> HttpResponse:
    book = get_object_or_404(Book, pk=pk)
    if request.method == 'POST':
        form = BookForm(request.POST, instance=book)
        if form.is_valid():
            form.save()
            messages.success(request, 'Book updated successfully!')
            if wants_fragment(request):
                return render(request, app_name_var + '/partials/book_list_items.html', {'books': Book.objects.all()})
            return redirect(reverse(app_name_var + ':book_list'))
    else:
        form = BookForm(instance=book)
    if wants_fragment(request):
        template = app_name_var + '/partials/book_form_partial.html'
    else:
        template = app_name_var + '/book_form.html'
    return render(request, template, {'form': form, 'action': 'Edit', 'book': book})


def book_delete(request: HttpRequest, pk: int) -> HttpResponse:
    book = get_object_or_404(Book, pk=pk)
    if request.method == 'POST':
        book.delete()
        messages.success(request, 'Book deleted successfully!')
        if wants_fragment(request):
            return HttpResponse('')
        return redirect(reverse(app_name_var + ':book_list'))
    return render(request, app_name_var + '/book_delete.html', {'book': book})


def book_search(request: HttpRequest) -> HttpResponse:
    query = request.GET.get('q', '').strip()
    books = Book.objects.filter(title__icontains=query) if query else Book.objects.all()
    if wants_fragment(request):
        return render(request, app_name_var + '/partials/book_list_items.html', {'books': books})
    # Reached by following the search URL directly, so serve the whole page
    # rather than a bare fragment with no stylesheet around it.
    return render(request, app_name_var + '/book_list.html', {'books': books, 'query': query})

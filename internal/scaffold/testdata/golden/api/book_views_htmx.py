from django.contrib import messages
from django.http import HttpResponse
from django.shortcuts import get_object_or_404
from django.shortcuts import redirect
from django.shortcuts import render
from django.urls import reverse

from .forms import BookForm
from .htmx import HtmxHttpRequest
from .htmx import wants_fragment
from .models import Book

app_name_var = 'catalog'


def book_list(request: HtmxHttpRequest) -> HttpResponse:
    books = Book.objects.search(request.GET.get('q', ''))
    if wants_fragment(request):
        return render(request, app_name_var + '/partials/book_list_items.html', {'books': books})
    return render(request, app_name_var + '/book_list.html', {'books': books})


def book_create(request: HtmxHttpRequest) -> HttpResponse:
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


def book_edit(request: HtmxHttpRequest, pk: int) -> HttpResponse:
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


def book_delete(request: HtmxHttpRequest, pk: int) -> HttpResponse:
    book = get_object_or_404(Book, pk=pk)
    if request.method == 'POST':
        book.delete()
        messages.success(request, 'Book deleted successfully!')
        if wants_fragment(request):
            return HttpResponse('')
        return redirect(reverse(app_name_var + ':book_list'))
    return render(request, app_name_var + '/book_delete.html', {'book': book})


def book_search(request: HtmxHttpRequest) -> HttpResponse:
    query = request.GET.get('q', '').strip()
    books = Book.objects.search(query)
    if wants_fragment(request):
        return render(request, app_name_var + '/partials/book_list_items.html', {'books': books})
    return render(request, app_name_var + '/book_list.html', {'books': books, 'query': query})

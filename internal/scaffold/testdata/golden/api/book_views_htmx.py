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


def book_list(request: HttpRequest) -> HttpResponse:
    books = Book.objects.all()
    if request.htmx:
        return render(request, app_name_var + '/partials/book_list_items.html', {'books': books})
    return render(request, app_name_var + '/book_list.html', {'books': books})


def book_create(request: HttpRequest) -> HttpResponse:
    if request.method == 'POST':
        form = BookForm(request.POST)
        if form.is_valid():
            form.save()
            messages.success(request, 'Book created successfully!')
            if request.htmx:
                return render(request, app_name_var + '/partials/book_list_items.html', {'books': Book.objects.all()})
            return redirect(reverse(app_name_var + ':book_list'))
    else:
        form = BookForm()
    template = app_name_var + '/partials/book_form_partial.html' if request.htmx else app_name_var + '/book_form.html'
    return render(request, template, {'form': form, 'action': 'Create'})


def book_edit(request: HttpRequest, pk: int) -> HttpResponse:
    book = get_object_or_404(Book, pk=pk)
    if request.method == 'POST':
        form = BookForm(request.POST, instance=book)
        if form.is_valid():
            form.save()
            messages.success(request, 'Book updated successfully!')
            if request.htmx:
                return render(request, app_name_var + '/partials/book_list_items.html', {'books': Book.objects.all()})
            return redirect(reverse(app_name_var + ':book_list'))
    else:
        form = BookForm(instance=book)
    template = app_name_var + '/partials/book_form_partial.html' if request.htmx else app_name_var + '/book_form.html'
    return render(request, template, {'form': form, 'action': 'Edit', 'book': book})


def book_delete(request: HttpRequest, pk: int) -> HttpResponse:
    book = get_object_or_404(Book, pk=pk)
    if request.method == 'POST':
        book.delete()
        messages.success(request, 'Book deleted successfully!')
        if request.htmx:
            return HttpResponse('')
        return redirect(reverse(app_name_var + ':book_list'))
    return render(request, app_name_var + '/book_delete.html', {'book': book})


def book_search(request: HttpRequest) -> HttpResponse:
    query = request.GET.get('q', '').strip()
    books = Book.objects.filter(title__icontains=query) if query else Book.objects.all()
    if request.htmx:
        return render(request, app_name_var + '/partials/book_list_items.html', {'books': books})
    # Reached by following the search URL directly, so serve the whole page
    # rather than a bare fragment with no stylesheet around it.
    return render(request, app_name_var + '/book_list.html', {'books': books, 'query': query})

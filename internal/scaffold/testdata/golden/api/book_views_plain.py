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
    return render(request, app_name_var + '/book_list.html', {'books': books})


def book_create(request: HttpRequest) -> HttpResponse:
    if request.method == 'POST':
        form = BookForm(request.POST)
        if form.is_valid():
            form.save()
            messages.success(request, 'Book created successfully!')
            return redirect(reverse(app_name_var + ':book_list'))
    else:
        form = BookForm()
    return render(request, app_name_var + '/book_form.html', {'form': form, 'action': 'Create'})


def book_edit(request: HttpRequest, pk: int) -> HttpResponse:
    book = get_object_or_404(Book, pk=pk)
    if request.method == 'POST':
        form = BookForm(request.POST, instance=book)
        if form.is_valid():
            form.save()
            messages.success(request, 'Book updated successfully!')
            return redirect(reverse(app_name_var + ':book_list'))
    else:
        form = BookForm(instance=book)
    return render(request, app_name_var + '/book_form.html', {'form': form, 'action': 'Edit', 'book': book})


def book_delete(request: HttpRequest, pk: int) -> HttpResponse:
    book = get_object_or_404(Book, pk=pk)
    if request.method == 'POST':
        book.delete()
        messages.success(request, 'Book deleted successfully!')
        return redirect(reverse(app_name_var + ':book_list'))
    return render(request, app_name_var + '/book_delete.html', {'book': book})

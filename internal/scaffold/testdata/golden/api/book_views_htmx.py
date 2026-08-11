from datetime import timedelta

from django.contrib import messages
from django.http import HttpResponse
from django.shortcuts import get_object_or_404, redirect, render
from django.utils import timezone
from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.response import Response

from .forms import BookForm
from .models import Book
from .serializers import BookSerializer

app_name_var = 'catalog'

def book_list(request):
    books = Book.objects.all().order_by('-created_at')
    if request.htmx:
        return render(request, app_name_var + '/partials/book_list_items.html', {'books': books})
    return render(request, app_name_var + '/book_list.html', {'books': books})

def book_create(request):
    if request.method == 'POST':
        form = BookForm(request.POST)
        if form.is_valid():
            form.save()
            messages.success(request, 'Book created successfully!')
            if request.htmx:
                books = Book.objects.all().order_by('-created_at')
                return render(request, app_name_var + '/partials/book_list_items.html', {'books': books})
            return redirect('/' + app_name_var + '/')
    else:
        form = BookForm()
    template = app_name_var + '/partials/book_form_partial.html' if request.htmx else app_name_var + '/book_form.html'
    return render(request, template, {'form': form, 'action': 'Create'})

def book_edit(request, book_id):
    book = get_object_or_404(Book, id=book_id)
    if request.method == 'POST':
        form = BookForm(request.POST, instance=book)
        if form.is_valid():
            form.save()
            messages.success(request, 'Book updated successfully!')
            if request.htmx:
                books = Book.objects.all().order_by('-created_at')
                return render(request, app_name_var + '/partials/book_list_items.html', {'books': books})
            return redirect('/' + app_name_var + '/')
    else:
        form = BookForm(instance=book)
    template = app_name_var + '/partials/book_form_partial.html' if request.htmx else app_name_var + '/book_form.html'
    return render(request, template, {'form': form, 'action': 'Edit', 'book': book})

def book_delete(request, book_id):
    book = get_object_or_404(Book, id=book_id)
    if request.method == 'POST':
        book.delete()
        messages.success(request, 'Book deleted successfully!')
        if request.htmx:
            return HttpResponse('')
        return redirect('/' + app_name_var + '/')
    return render(request, app_name_var + '/book_delete.html', {'book': book})

def book_search(request):
    query = request.GET.get('q', '').strip()
    if query:
        books = Book.objects.filter(title__icontains=query).order_by('-created_at')
    else:
        books = Book.objects.all().order_by('-created_at')
    return render(request, app_name_var + '/partials/book_list_items.html', {'books': books})

class BookViewSet(viewsets.ModelViewSet):
    queryset = Book.objects.all()
    serializer_class = BookSerializer

    @action(detail=False, methods=['get'])
    def recent(self, request):
        recent_books = Book.objects.filter(
            created_at__gte=timezone.now() - timedelta(days=30)
        )
        serializer = self.get_serializer(recent_books, many=True)
        return Response(serializer.data)

package ui

// generateAppURLs returns the content of the app's urls.py file.
// When htmx is true, an additional search URL pattern is included.
func generateAppURLs(appName string, htmx bool) string {
	if htmx {
		return `from django.urls import path

from . import views

app_name = '` + appName + `'

urlpatterns = [
    path('', views.book_list, name='book_list'),
    path('create/', views.book_create, name='book_create'),
    path('edit/<int:book_id>/', views.book_edit, name='book_edit'),
    path('delete/<int:book_id>/', views.book_delete, name='book_delete'),
    path('search/', views.book_search, name='book_search'),
]
`
	}
	return `from django.urls import path

from . import views

app_name = '` + appName + `'

urlpatterns = [
    path('', views.book_list, name='book_list'),
    path('create/', views.book_create, name='book_create'),
    path('edit/<int:book_id>/', views.book_edit, name='book_edit'),
    path('delete/<int:book_id>/', views.book_delete, name='book_delete'),
]
`
}

// generateBookSerializer returns the content of serializers.py.
func generateBookSerializer(appName string) string {
	return `from rest_framework import serializers

from .models import Book

class BookSerializer(serializers.ModelSerializer):
    class Meta:
        model = Book
        fields = '__all__'
        read_only_fields = ('created_at', 'updated_at')
`
}

// generateBookForms returns the content of forms.py.
// When tailwind is true, Tailwind CSS classes are applied to form widgets.
func generateBookForms(appName string, tailwind bool) string {
	twInputClass := ""
	if tailwind {
		twInputClass = "w-full rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-white placeholder:text-white/35 outline-none transition focus:border-white/30 focus:bg-white/[0.05] focus:ring-2 focus:ring-white/10"
	}

	return `from django import forms

from .models import Book

class BookForm(forms.ModelForm):
    class Meta:
        model = Book
        fields = ['title', 'author', 'isbn', 'publication_date', 'price']
        widgets = {
            'title': forms.TextInput(attrs={'class': '` + twInputClass + `', 'placeholder': 'Atomic Habits'}),
            'author': forms.TextInput(attrs={'class': '` + twInputClass + `', 'placeholder': 'James Clear'}),
            'isbn': forms.TextInput(attrs={'class': '` + twInputClass + `', 'placeholder': '9780735211292'}),
            'publication_date': forms.DateInput(attrs={'class': '` + twInputClass + `', 'type': 'date'}),
            'price': forms.NumberInput(attrs={'class': '` + twInputClass + `', 'placeholder': '24.99', 'step': '0.01'}),
        }
`
}

// generateBookAPI returns the content of api.py with router configuration.
func generateBookAPI(appName string) string {
	return `from rest_framework import routers

from .views import BookViewSet

router = routers.DefaultRouter()
router.register(r'books', BookViewSet, basename='book')

urlpatterns = router.urls
`
}

// generateBookModel returns the content of models.py with the Book model.
func generateBookModel(appName string) string {
	return `from django.db import models

class Book(models.Model):
    title = models.CharField(max_length=200)
    author = models.CharField(max_length=100)
    isbn = models.CharField(max_length=13, unique=True)
    publication_date = models.DateField()
    price = models.DecimalField(max_digits=10, decimal_places=2)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return self.title
`
}

// generateBookViews returns the content of views.py.
// When htmx is true, HTMX-aware view logic is included (partials, search, HttpResponse).
func generateBookViews(appName string, htmx bool) string {
	if htmx {
		return `from datetime import timedelta

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

app_name_var = '` + appName + `'

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
`
	}
	return `from datetime import timedelta

from django.contrib import messages
from django.shortcuts import get_object_or_404, redirect, render
from django.utils import timezone
from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.response import Response

from .forms import BookForm
from .models import Book
from .serializers import BookSerializer

app_name_var = '` + appName + `'

def book_list(request):
    books = Book.objects.all().order_by('-created_at')
    return render(request, app_name_var + '/book_list.html', {'books': books})

def book_create(request):
    if request.method == 'POST':
        form = BookForm(request.POST)
        if form.is_valid():
            form.save()
            messages.success(request, 'Book created successfully!')
            return redirect('/' + app_name_var + '/')
    else:
        form = BookForm()
    return render(request, app_name_var + '/book_form.html', {'form': form, 'action': 'Create'})

def book_edit(request, book_id):
    book = get_object_or_404(Book, id=book_id)
    if request.method == 'POST':
        form = BookForm(request.POST, instance=book)
        if form.is_valid():
            form.save()
            messages.success(request, 'Book updated successfully!')
            return redirect('/' + app_name_var + '/')
    else:
        form = BookForm(instance=book)
    return render(request, app_name_var + '/book_form.html', {'form': form, 'action': 'Edit', 'book': book})

def book_delete(request, book_id):
    book = get_object_or_404(Book, id=book_id)
    if request.method == 'POST':
        book.delete()
        messages.success(request, 'Book deleted successfully!')
        return redirect('/' + app_name_var + '/')
    return render(request, app_name_var + '/book_delete.html', {'book': book})

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
`
}

// generateAdminPy returns a simple admin.py registering the Book model.
func generateAdminPy(appName string) string {
	return `from django.contrib import admin

from .models import Book


@admin.register(Book)
class BookAdmin(admin.ModelAdmin):
    list_display = ("title", "author", "isbn", "publication_date", "price", "created_at")
    search_fields = ("title", "author", "isbn")
    list_filter = ("publication_date", "created_at", "updated_at")
    ordering = ("-created_at",)
    readonly_fields = ("created_at", "updated_at")
`
}

// generateSampleData returns the content of the create_sample_data management command.
func generateSampleData(appName string) string {
	return `from datetime import date

from django.core.management.base import BaseCommand

from ` + appName + `.models import Book

class Command(BaseCommand):
    help = 'Creates sample book data'

    def handle(self, *args, **options):
        books = [
            {
                'title': 'Django for Beginners',
                'author': 'William Vincent',
                'isbn': '9781735467200',
                'publication_date': date(2022, 1, 1),
                'price': 39.99
            },
            {
                'title': 'Two Scoops of Django',
                'author': 'Daniel Roy Greenfeld',
                'isbn': '9780692915738',
                'publication_date': date(2021, 5, 15),
                'price': 49.99
            }
        ]

        for book_data in books:
            isbn = book_data['isbn']
            defaults = {key: value for key, value in book_data.items() if key != 'isbn'}
            Book.objects.update_or_create(isbn=isbn, defaults=defaults)

        self.stdout.write(self.style.SUCCESS('Sample data created successfully!'))
`
}

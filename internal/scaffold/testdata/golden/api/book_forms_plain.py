from django import forms

from .models import Book

class BookForm(forms.ModelForm):
    class Meta:
        model = Book
        fields = ['title', 'author', 'isbn', 'publication_date', 'price']
        widgets = {
            'title': forms.TextInput(attrs={'class': '', 'placeholder': 'Atomic Habits'}),
            'author': forms.TextInput(attrs={'class': '', 'placeholder': 'James Clear'}),
            'isbn': forms.TextInput(attrs={'class': '', 'placeholder': '9780735211292'}),
            'publication_date': forms.DateInput(attrs={'class': '', 'type': 'date'}),
            'price': forms.NumberInput(attrs={'class': '', 'placeholder': '24.99', 'step': '0.01'}),
        }

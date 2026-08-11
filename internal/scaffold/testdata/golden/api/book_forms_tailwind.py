from django import forms

from .models import Book

# Shared widget styling. Kept as a module constant rather than repeated inline:
# the class list is ~250 characters, and inlining it in every widget produced
# 300-character lines that ruff rejects with E501 and cannot autofix.
INPUT_CLASS = (
    "w-full rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3 "
    "text-sm text-white placeholder:text-white/35 outline-none transition "
    "focus:border-white/30 focus:bg-white/[0.05] focus:ring-2 "
    "focus:ring-white/10"
)


class BookForm(forms.ModelForm):
    class Meta:
        model = Book
        fields = ['title', 'author', 'isbn', 'publication_date', 'price']
        widgets = {
            'title': forms.TextInput(attrs={'class': INPUT_CLASS, 'placeholder': 'Atomic Habits'}),
            'author': forms.TextInput(attrs={'class': INPUT_CLASS, 'placeholder': 'James Clear'}),
            'isbn': forms.TextInput(attrs={'class': INPUT_CLASS, 'placeholder': '9780735211292'}),
            'publication_date': forms.DateInput(attrs={'class': INPUT_CLASS, 'type': 'date'}),
            'price': forms.NumberInput(attrs={'class': INPUT_CLASS, 'placeholder': '24.99', 'step': '0.01'}),
        }

from django import forms

from .models import Book

# Kept as a module constant rather than repeated inline: inlining it in every
# widget produced 300-character lines that ruff rejects with E501 and cannot
# autofix, aborting the scaffold's formatting pass.
INPUT_CLASS = (
    "mt-2 w-full rounded-[6px] border border-rule bg-ground px-3 py-1.5 "
    "text-heading outline-none transition-colors duration-300 "
    "focus:border-accent-border"
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

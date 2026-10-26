from django import forms

from .models import Book

INPUT_CLASS = ""


class BookForm(forms.ModelForm[Book]):
    class Meta:
        model = Book
        fields = ["title", "author", "isbn", "publication_date", "price"]
        widgets = {
            "title": forms.TextInput(attrs={"class": INPUT_CLASS, "placeholder": "Atomic Habits"}),
            "author": forms.TextInput(attrs={"class": INPUT_CLASS, "placeholder": "James Clear"}),
            "isbn": forms.TextInput(attrs={"class": INPUT_CLASS, "placeholder": "9780735211292"}),
            "publication_date": forms.DateInput(attrs={"class": INPUT_CLASS, "type": "date"}),
            "price": forms.NumberInput(attrs={"class": INPUT_CLASS, "placeholder": "24.99", "step": "0.01"}),
        }

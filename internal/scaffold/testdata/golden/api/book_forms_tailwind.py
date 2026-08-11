from django import forms

from .models import Book

class BookForm(forms.ModelForm):
    class Meta:
        model = Book
        fields = ['title', 'author', 'isbn', 'publication_date', 'price']
        widgets = {
            'title': forms.TextInput(attrs={'class': 'w-full rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-white placeholder:text-white/35 outline-none transition focus:border-white/30 focus:bg-white/[0.05] focus:ring-2 focus:ring-white/10', 'placeholder': 'Atomic Habits'}),
            'author': forms.TextInput(attrs={'class': 'w-full rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-white placeholder:text-white/35 outline-none transition focus:border-white/30 focus:bg-white/[0.05] focus:ring-2 focus:ring-white/10', 'placeholder': 'James Clear'}),
            'isbn': forms.TextInput(attrs={'class': 'w-full rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-white placeholder:text-white/35 outline-none transition focus:border-white/30 focus:bg-white/[0.05] focus:ring-2 focus:ring-white/10', 'placeholder': '9780735211292'}),
            'publication_date': forms.DateInput(attrs={'class': 'w-full rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-white placeholder:text-white/35 outline-none transition focus:border-white/30 focus:bg-white/[0.05] focus:ring-2 focus:ring-white/10', 'type': 'date'}),
            'price': forms.NumberInput(attrs={'class': 'w-full rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-white placeholder:text-white/35 outline-none transition focus:border-white/30 focus:bg-white/[0.05] focus:ring-2 focus:ring-white/10', 'placeholder': '24.99', 'step': '0.01'}),
        }

from datetime import date
from decimal import Decimal
from typing import Any

from django.core.management.base import BaseCommand
from django.db import transaction

from catalog.models import Book

BOOKS = [
    {
        'title': 'Django for Beginners',
        'author': 'William Vincent',
        'isbn': '9781735467200',
        'publication_date': date(2022, 1, 1),
        'price': Decimal('39.99'),
    },
    {
        'title': 'Two Scoops of Django',
        'author': 'Daniel Roy Greenfeld',
        'isbn': '9780692915738',
        'publication_date': date(2021, 5, 15),
        'price': Decimal('49.99'),
    },
]


class Command(BaseCommand):
    help = 'Creates sample book data'

    @transaction.atomic
    def handle(self, *args: Any, **options: Any) -> None:
        for book_data in BOOKS:
            defaults = {key: value for key, value in book_data.items() if key != 'isbn'}
            Book.objects.update_or_create(isbn=book_data['isbn'], defaults=defaults)

        self.stdout.write(self.style.SUCCESS('Sample data created successfully!'))

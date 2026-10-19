from datetime import date
from decimal import Decimal
from typing import TypedDict

from django.core.management.base import BaseCommand

from catalog.models import Book


class BookSeed(TypedDict):
    title: str
    author: str
    isbn: str
    publication_date: date
    price: Decimal


BOOKS: list[BookSeed] = [
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

    def handle(self, *args: str, **options: object) -> None:
        Book.objects.bulk_create(
            [Book(**book) for book in BOOKS],
            update_conflicts=True,
            update_fields=['title', 'author', 'publication_date', 'price'],
            unique_fields=['isbn'],
        )

        self.stdout.write(self.style.SUCCESS(f'Seeded {len(BOOKS)} books.'))

from datetime import date

from django.core.management.base import BaseCommand

from catalog.models import Book

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

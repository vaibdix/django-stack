from datetime import date
from datetime import timedelta

from django.core.validators import MinValueValidator
from django.core.validators import RegexValidator
from django.db import models
from django.db.models import Q
from django.urls import reverse
from django.utils import timezone

ISBN_VALIDATOR = RegexValidator(r'^\d{13}$', 'ISBN must be exactly 13 digits.')

RECENT_WINDOW_DAYS = 30

EARLIEST_PUBLICATION_DATE = date(1450, 1, 1)


class BookQuerySet(models.QuerySet['Book']):
    """The questions this project asks about books, asked in one place."""

    def search(self, term: str) -> 'BookQuerySet':
        cleaned = term.strip()
        if not cleaned:
            return self
        return self.filter(
            Q(title__icontains=cleaned)
            | Q(author__icontains=cleaned)
            | Q(isbn__icontains=cleaned),
        )

    def recent(self, days: int = RECENT_WINDOW_DAYS) -> 'BookQuerySet':
        return self.filter(created_at__gte=timezone.now() - timedelta(days=days))


class BookManager(models.Manager['Book']):
    """Re-exposes the queryset methods so Book.objects.search() type-checks."""

    def get_queryset(self) -> BookQuerySet:
        return BookQuerySet(self.model, using=self._db)

    def search(self, term: str) -> BookQuerySet:
        return self.get_queryset().search(term)

    def recent(self, days: int = RECENT_WINDOW_DAYS) -> BookQuerySet:
        return self.get_queryset().recent(days)


class Book(models.Model):
    title = models.CharField(max_length=200, db_index=True, help_text='The full title, as printed on the cover.')
    author = models.CharField(max_length=100, help_text='Primary author. One name per book in this example.')
    isbn = models.CharField(
        max_length=13,
        unique=True,
        validators=[ISBN_VALIDATOR],
        help_text='Thirteen digits, no hyphens.',
    )
    publication_date = models.DateField(help_text='The date this edition was published.')
    price = models.DecimalField(
        max_digits=10,
        decimal_places=2,
        validators=[MinValueValidator(0)],
        help_text='Retail price. Never negative.',
    )
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    objects = BookManager()

    class Meta:
        ordering = ('-created_at', '-pk')
        indexes = [models.Index(fields=('-created_at', '-id'))]
        constraints = [
            models.CheckConstraint(
                condition=Q(price__gte=0),
                name='book_price_not_negative',
            ),
            models.CheckConstraint(
                condition=Q(publication_date__gte=EARLIEST_PUBLICATION_DATE),
                name='book_publication_date_plausible',
            ),
        ]

    def __str__(self) -> str:
        return self.title

    def get_absolute_url(self) -> str:
        return reverse('catalog:book_edit', args=(self.pk,))

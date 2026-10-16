from datetime import date
from datetime import timedelta

from django.core.validators import MinValueValidator
from django.db import models
from django.db.models import Q
from django.urls import reverse
from django.utils import timezone

from core.models import TimeStampedModel

from .fields import ISBN_INPUT_MAX_LENGTH
from .fields import ISBN_VALIDATOR
from .fields import ISBNField
from .fields import normalise_isbn
from .fields import validate_not_in_future

RECENT_WINDOW_DAYS = 30

EARLIEST_PUBLICATION_DATE = date(1450, 1, 1)

PUBLICATION_DATE_TOO_EARLY = 'A book cannot be published before 1450.'

NEGATIVE_PRICE = 'A price cannot be negative.'


class BookQuerySet(models.QuerySet['Book']):
    def search(self, term: str) -> 'BookQuerySet':
        cleaned = term.strip()
        if not cleaned:
            return self
        return self.filter(
            Q(title__icontains=cleaned)
            | Q(author__icontains=cleaned)
            | Q(isbn__icontains=normalise_isbn(cleaned)),
        )

    def recent(self, days: int = RECENT_WINDOW_DAYS) -> 'BookQuerySet':
        return self.filter(created_at__gte=timezone.now() - timedelta(days=days))


class Book(TimeStampedModel):
    title = models.CharField(max_length=200, help_text='The full title, as printed on the cover.')
    author = models.CharField(max_length=100, help_text='Primary author. One name per book in this example.')
    isbn = ISBNField(
        max_length=ISBN_INPUT_MAX_LENGTH,
        unique=True,
        validators=[ISBN_VALIDATOR],
        help_text='Thirteen digits. Hyphens and spaces are accepted and removed.',
    )
    publication_date = models.DateField(
        validators=[MinValueValidator(EARLIEST_PUBLICATION_DATE, PUBLICATION_DATE_TOO_EARLY), validate_not_in_future],
        help_text='The date this edition was published. Never in the future.',
    )
    price = models.DecimalField(
        max_digits=10,
        decimal_places=2,
        validators=[MinValueValidator(0, NEGATIVE_PRICE)],
        help_text='Retail price. Never negative.',
    )
    objects = BookQuerySet.as_manager()

    class Meta:
        ordering = ('-created_at', '-pk')
        indexes = [models.Index(fields=('-created_at', '-id'))]
        constraints = [
            models.CheckConstraint(
                condition=Q(price__gte=0),
                name='book_price_not_negative',
                violation_error_message=NEGATIVE_PRICE,
            ),
            models.CheckConstraint(
                condition=Q(publication_date__gte=EARLIEST_PUBLICATION_DATE),
                name='book_publication_date_plausible',
                violation_error_message=PUBLICATION_DATE_TOO_EARLY,
            ),
        ]

    def __str__(self) -> str:
        return self.title

    def get_absolute_url(self) -> str:
        return reverse('catalog:book_edit', args=(self.pk,))

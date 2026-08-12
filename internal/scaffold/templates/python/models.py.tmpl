from django.core.validators import MinValueValidator
from django.core.validators import RegexValidator
from django.db import models

# Kept in step with schemas.py, which validates the same shape at the API edge.
# If only one of them changes, the API and the admin start disagreeing about
# what a valid book is.
ISBN_VALIDATOR = RegexValidator(r'^\d{13}$', 'ISBN must be exactly 13 digits.')


class Book(models.Model):
    title = models.CharField(max_length=200, db_index=True)
    author = models.CharField(max_length=100)
    isbn = models.CharField(max_length=13, unique=True, validators=[ISBN_VALIDATOR])
    publication_date = models.DateField()
    price = models.DecimalField(
        max_digits=10,
        decimal_places=2,
        validators=[MinValueValidator(0)],
    )
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        ordering = ('-created_at',)
        indexes = [models.Index(fields=('-created_at',))]

    def __str__(self) -> str:
        return self.title

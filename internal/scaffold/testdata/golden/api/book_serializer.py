from datetime import date

from django.utils import timezone
from rest_framework import serializers

from catalog.models import Book

ISBN_DIGITS = 13


class BookSerializer(serializers.ModelSerializer[Book]):
    class Meta:
        model = Book
        fields = (
            'id',
            'title',
            'author',
            'isbn',
            'publication_date',
            'price',
            'created_at',
            'updated_at',
        )
        read_only_fields = ('id', 'created_at', 'updated_at')

    def validate_isbn(self, value: str) -> str:
        normalised = value.replace('-', '').replace(' ', '')
        if not normalised.isdigit() or len(normalised) != ISBN_DIGITS:
            message = 'ISBN must be exactly 13 digits, with or without hyphens.'
            raise serializers.ValidationError(message)
        return normalised

    def validate_publication_date(self, value: date) -> date:
        if value > timezone.localdate():
            message = 'A book cannot be published in the future.'
            raise serializers.ValidationError(message)
        return value

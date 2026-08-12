from rest_framework import serializers

from .models import Book


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

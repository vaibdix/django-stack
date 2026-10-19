from rest_framework import serializers

from catalog.fields import ISBNField
from catalog.fields import normalise_isbn
from catalog.models import Book


class ISBNSerializerField(serializers.CharField):
    def to_internal_value(self, data: str) -> str:
        return normalise_isbn(super().to_internal_value(data))


class BookSerializer(serializers.ModelSerializer[Book]):
    serializer_field_mapping = {
        **serializers.ModelSerializer.serializer_field_mapping,
        ISBNField: ISBNSerializerField,
    }

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

from datetime import timedelta

from django.utils import timezone
from rest_framework import filters
from rest_framework import routers
from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.request import Request
from rest_framework.response import Response

from .models import Book
from .serializers import BookSerializer


class BookViewSet(viewsets.ModelViewSet[Book]):
    queryset = Book.objects.all()
    serializer_class = BookSerializer
    filter_backends = (filters.SearchFilter, filters.OrderingFilter)
    search_fields = ('title', 'author', 'isbn')
    ordering_fields = ('title', 'publication_date', 'price', 'created_at')

    @action(detail=False, methods=['get'])
    def recent(self, request: Request) -> Response:
        recent_books = self.filter_queryset(
            self.get_queryset().filter(created_at__gte=timezone.now() - timedelta(days=30))
        )
        page = self.paginate_queryset(recent_books)
        if page is not None:
            return self.get_paginated_response(self.get_serializer(page, many=True).data)
        return Response(self.get_serializer(recent_books, many=True).data)


router = routers.DefaultRouter()
router.register(r'books', BookViewSet, basename='book')

urlpatterns = router.urls

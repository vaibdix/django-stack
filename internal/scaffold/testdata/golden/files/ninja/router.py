from django.http import HttpRequest
from django.shortcuts import get_object_or_404
from ninja import Router
from ninja import Status
from ninja.pagination import PageNumberPagination
from ninja.pagination import paginate
from ninja.security import django_auth

from catalog.api.schemas import BookIn
from catalog.api.schemas import BookOut
from catalog.models import Book
from catalog.models import BookQuerySet

PAGE_SIZE = 20

router = Router(tags=['books'])


@router.get('', response=list[BookOut])
@paginate(PageNumberPagination, page_size=PAGE_SIZE)
def list_books(request: HttpRequest, search: str = '') -> BookQuerySet:
    return Book.objects.search(search)


@router.get('/recent', response=list[BookOut])
@paginate(PageNumberPagination, page_size=PAGE_SIZE)
def recent_books(request: HttpRequest, search: str = '') -> BookQuerySet:
    return Book.objects.search(search).recent()


@router.get('/{int:book_id}', response=BookOut)
def get_book(request: HttpRequest, book_id: int) -> Book:
    return get_object_or_404(Book, pk=book_id)


def validated_save(book: Book) -> Book:
    book.full_clean()
    book.save()
    return book


@router.post('', response={201: BookOut}, auth=django_auth)
def create_book(request: HttpRequest, payload: BookIn) -> Status[Book]:
    return Status(201, validated_save(Book(**payload.model_dump())))


@router.put('/{int:book_id}', response=BookOut, auth=django_auth)
def update_book(request: HttpRequest, book_id: int, payload: BookIn) -> Book:
    book = get_object_or_404(Book, pk=book_id)
    for field, value in payload.model_dump().items():
        setattr(book, field, value)
    return validated_save(book)


@router.delete('/{int:book_id}', response={204: None}, auth=django_auth)
def delete_book(request: HttpRequest, book_id: int) -> Status[None]:
    get_object_or_404(Book, pk=book_id).delete()
    return Status(204, None)

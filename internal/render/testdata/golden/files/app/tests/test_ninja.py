from datetime import date
from decimal import Decimal

from django.contrib.auth import get_user_model
from django.test import TestCase
from django.urls import reverse

from catalog.models import Book

TEST_PASSWORD = "not-a-real-password"  # noqa: S105

NEW_BOOK = {
    "title": "Architecture Patterns with Python",
    "author": "Harry Percival",
    "isbn": "9781492052203",
    "publication_date": "2020-03-05",
    "price": "54.99",
}


class CatalogAPITests(TestCase):
    def setUp(self) -> None:
        self.book = Book.objects.create(
            title="Fluent Python",
            author="Luciano Ramalho",
            isbn="9781492056355",
            publication_date=date(2022, 4, 1),
            price=Decimal("64.99"),
        )

    def sign_in(self) -> None:
        user = get_user_model().objects.create_user(username="editor", password=TEST_PASSWORD)
        self.client.force_login(user)

    def test_anonymous_callers_can_read(self) -> None:
        response = self.client.get(reverse("api:list_books"))
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["count"], 1)

    def test_a_single_book_is_found_by_id(self) -> None:
        response = self.client.get(reverse("api:get_book", args=[self.book.pk]))
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["isbn"], "9781492056355")

        response = self.client.get(reverse("api:get_book", args=[self.book.pk + 1000]))
        self.assertEqual(response.status_code, 404)

    def test_anonymous_callers_cannot_write(self) -> None:
        response = self.client.post(reverse("api:create_book"), NEW_BOOK, content_type="application/json")
        self.assertEqual(response.status_code, 401)
        self.assertFalse(Book.objects.filter(isbn=NEW_BOOK["isbn"]).exists())

    def test_authenticated_callers_can_create(self) -> None:
        self.sign_in()
        response = self.client.post(reverse("api:create_book"), NEW_BOOK, content_type="application/json")
        self.assertEqual(response.status_code, 201)
        self.assertTrue(Book.objects.filter(isbn=NEW_BOOK["isbn"]).exists())

    def test_authenticated_callers_can_delete(self) -> None:
        self.sign_in()
        response = self.client.delete(reverse("api:delete_book", args=[self.book.pk]))
        self.assertEqual(response.status_code, 204)
        self.assertFalse(Book.objects.filter(pk=self.book.pk).exists())

    def test_recent_books_are_routed(self) -> None:
        response = self.client.get(reverse("api:recent_books"))
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["count"], 1)

    def test_search_matches_title_author_and_isbn(self) -> None:
        for term in ("fluent", "ramalho", "9781492"):
            response = self.client.get(reverse("api:list_books"), {"search": term})
            self.assertEqual(response.json()["count"], 1, term)

        response = self.client.get(reverse("api:list_books"), {"search": "nothing matches this"})
        self.assertEqual(response.json()["count"], 0)

    def test_a_hyphenated_isbn_is_normalised(self) -> None:
        self.sign_in()
        payload = {**NEW_BOOK, "isbn": "978-1-2345-6789-7"}
        response = self.client.post(reverse("api:create_book"), payload, content_type="application/json")
        self.assertEqual(response.status_code, 201)
        self.assertEqual(response.json()["isbn"], "9781234567897")

    def test_a_book_cannot_be_published_in_the_future(self) -> None:
        self.sign_in()
        payload = {**NEW_BOOK, "publication_date": "2999-01-01"}
        response = self.client.post(reverse("api:create_book"), payload, content_type="application/json")
        self.assertEqual(response.status_code, 422)
        self.assertFalse(Book.objects.filter(isbn=NEW_BOOK["isbn"]).exists())

    def test_a_book_cannot_predate_printing(self) -> None:
        self.sign_in()
        payload = {**NEW_BOOK, "publication_date": "1400-01-01"}
        response = self.client.post(reverse("api:create_book"), payload, content_type="application/json")
        self.assertEqual(response.status_code, 422)
        self.assertFalse(Book.objects.filter(isbn=NEW_BOOK["isbn"]).exists())

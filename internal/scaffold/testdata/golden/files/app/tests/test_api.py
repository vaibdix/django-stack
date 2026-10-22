from datetime import date
from decimal import Decimal

from django.contrib.auth import get_user_model
from django.test import TestCase
from django.urls import reverse

from catalog.models import Book

TEST_PASSWORD = 'not-a-real-password'  # noqa: S105

NEW_BOOK = {
    'title': 'Architecture Patterns with Python',
    'author': 'Harry Percival',
    'isbn': '9781492052203',
    'publication_date': '2020-03-05',
    'price': '54.99',
}


class CatalogAPITests(TestCase):
    def setUp(self) -> None:
        self.book = Book.objects.create(
            title='Fluent Python',
            author='Luciano Ramalho',
            isbn='9781492056355',
            publication_date=date(2022, 4, 1),
            price=Decimal('64.99'),
        )

    def sign_in(self) -> None:
        user = get_user_model().objects.create_user(username='editor', password=TEST_PASSWORD)
        self.client.force_login(user)

    def test_anonymous_callers_can_read(self) -> None:
        response = self.client.get(reverse('book-list'))
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()['count'], 1)

    def test_anonymous_callers_cannot_write(self) -> None:
        response = self.client.post(reverse('book-list'), NEW_BOOK)
        self.assertIn(response.status_code, (401, 403))
        self.assertFalse(Book.objects.filter(isbn=NEW_BOOK['isbn']).exists())

    def test_authenticated_callers_can_create(self) -> None:
        self.sign_in()
        response = self.client.post(reverse('book-list'), NEW_BOOK)
        self.assertEqual(response.status_code, 201)
        self.assertTrue(Book.objects.filter(isbn=NEW_BOOK['isbn']).exists())

    def test_recent_action_is_routed(self) -> None:
        response = self.client.get(reverse('book-recent'))
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()['count'], 1)

    def test_search_matches_title_author_and_isbn(self) -> None:
        for term in ('fluent', 'ramalho', '9781492'):
            response = self.client.get(reverse('book-list'), {'search': term})
            self.assertEqual(response.json()['count'], 1, term)

        response = self.client.get(reverse('book-list'), {'search': 'nothing matches this'})
        self.assertEqual(response.json()['count'], 0)

    def test_a_hyphenated_isbn_is_normalised(self) -> None:
        self.sign_in()
        response = self.client.post(reverse('book-list'), {**NEW_BOOK, 'isbn': '978-1-4920-5220-3'})
        self.assertEqual(response.status_code, 201)
        self.assertEqual(response.json()['isbn'], NEW_BOOK['isbn'])

    def test_a_book_cannot_be_published_in_the_future(self) -> None:
        self.sign_in()
        response = self.client.post(reverse('book-list'), {**NEW_BOOK, 'publication_date': '2999-01-01'})
        self.assertEqual(response.status_code, 400)
        self.assertIn('publication_date', response.json())

    def test_a_book_cannot_predate_printing(self) -> None:
        self.sign_in()
        response = self.client.post(reverse('book-list'), {**NEW_BOOK, 'publication_date': '1400-01-01'})
        self.assertEqual(response.status_code, 400)
        self.assertEqual(response.json()['publication_date'], ['A book cannot be published before 1450.'])

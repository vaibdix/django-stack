from datetime import date
from datetime import datetime
from decimal import Decimal

from ninja import Schema
from pydantic import ConfigDict


class BookIn(Schema):
    model_config = ConfigDict(extra="forbid", str_strip_whitespace=True)

    title: str
    author: str
    isbn: str
    publication_date: date
    price: Decimal


class BookOut(Schema):
    id: int
    title: str
    author: str
    isbn: str
    publication_date: date
    price: Decimal
    created_at: datetime
    updated_at: datetime

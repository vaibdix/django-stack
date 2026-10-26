from django.urls import path

from . import views

app_name = "catalog"

urlpatterns = [
    path("", views.BookListView.as_view(), name="book_list"),
    path("create/", views.BookCreateView.as_view(), name="book_create"),
    path("edit/<int:pk>/", views.BookUpdateView.as_view(), name="book_edit"),
    path("delete/<int:pk>/", views.BookDeleteView.as_view(), name="book_delete"),
]

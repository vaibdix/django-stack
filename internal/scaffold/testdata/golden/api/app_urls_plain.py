from django.urls import path

from . import views

app_name = 'catalog'

urlpatterns = [
    path('', views.book_list, name='book_list'),
    path('create/', views.book_create, name='book_create'),
    path('edit/<int:book_id>/', views.book_edit, name='book_edit'),
    path('delete/<int:book_id>/', views.book_delete, name='book_delete'),
]

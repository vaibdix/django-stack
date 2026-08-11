from rest_framework import routers

from .views import BookViewSet

router = routers.DefaultRouter()
router.register(r'books', BookViewSet, basename='book')

urlpatterns = router.urls

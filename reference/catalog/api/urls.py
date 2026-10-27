from rest_framework import routers

from catalog.api.views import BookViewSet

router = routers.DefaultRouter()
router.register(r"books", BookViewSet, basename="book")

urlpatterns = router.urls

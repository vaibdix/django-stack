from django.contrib.auth.models import AbstractUser
from django.db import models
from django.db.models.functions import Lower


class User(AbstractUser):
    email = models.EmailField()

    class Meta(AbstractUser.Meta):
        constraints = [
            models.UniqueConstraint(
                Lower('email'),
                name='accounts_user_email_ci_unique',
                violation_error_message='A user with this email address already exists.',
            ),
        ]

    def __str__(self) -> str:
        return self.get_username()

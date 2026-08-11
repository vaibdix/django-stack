# Superuser Setup

This project does not create a default Django superuser automatically.

Use the project virtual environment when creating your first admin account.

## macOS / Linux

```
./.venv/bin/python manage.py createsuperuser
```

## Windows

```
.venv\Scripts\python.exe manage.py createsuperuser
```

After the account is created, open `/admin/` and sign in with the credentials you chose.

Important:
- Prefer the commands above for manual Django management tasks.
- Do not use plain `python manage.py ...` unless you have activated the project venv first.

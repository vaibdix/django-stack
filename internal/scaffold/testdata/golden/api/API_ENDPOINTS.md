# my-shop API Endpoints

## API Documentation

| URL | Description |
|-----|-------------|
| [http://localhost:8000/api/schema/swagger-ui/](http://localhost:8000/api/schema/swagger-ui/) | Swagger UI (interactive API docs) |
| [http://localhost:8000/api/schema/redoc/](http://localhost:8000/api/schema/redoc/) | ReDoc (alternative API docs) |
| [http://localhost:8000/api/schema/](http://localhost:8000/api/schema/) | Raw OpenAPI schema (JSON/YAML) |

## Authentication

The API is **read-only for anonymous callers**. `GET` works without signing in; `POST`, `PUT`, `PATCH` and `DELETE` require an authenticated user, so the browsable API shows no create or edit forms until you log in.

Create a superuser (see SUPERUSER_SETUP.md), then sign in at the URL below and the forms appear on every endpoint.

| URL | Description |
|-----|-------------|
| [http://localhost:8000/api-auth/login/](http://localhost:8000/api-auth/login/) | Log in to the browsable API |
| [http://localhost:8000/api-auth/logout/](http://localhost:8000/api-auth/logout/) | Log out |

To make the demo API writable without signing in, change `DEFAULT_PERMISSION_CLASSES` in settings to `rest_framework.permissions.AllowAny`.

## catalog API

| URL | Description |
|-----|-------------|
| [http://localhost:8000/api/v1/](http://localhost:8000/api/v1/) | catalog API endpoints |

## Admin

| URL | Description |
|-----|-------------|
| [http://localhost:8000/admin/](http://localhost:8000/admin/) | Django admin interface |

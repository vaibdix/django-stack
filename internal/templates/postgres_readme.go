package templates

func PostgresReadmeTemplate() string {
	return `# PostgreSQL Setup Guide

This project was configured to use PostgreSQL but requires manual setup.

## Prerequisites

1. **Install PostgreSQL**
   - Download from: https://www.postgresql.org/download/
   - Or use a PostgreSQL service (Supabase, Railway, Neon, etc.)

2. **Install psycopg2-binary**
   # Activate virtual environment first, then:
   ./.venv/bin/python -m pip install psycopg2-binary
   # Or: .venv\Scripts\python.exe -m pip install psycopg2-binary (Windows)

## Configuration

### Option 1: Local PostgreSQL

1. Create a database:
   createdb myproject_db

2. Update your settings.py with your PostgreSQL credentials:

` + "```" + `python
DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.postgresql',
        'NAME': 'myproject_db',      # Database name
        'USER': 'postgres',           # Your PostgreSQL username
        'PASSWORD': 'your_password',  # Your PostgreSQL password
        'HOST': 'localhost',
        'PORT': '5432',
    }
}
` + "```" + `

3. Run migrations:
   ./.venv/bin/python manage.py migrate
   # Or: .venv\\Scripts\\python.exe manage.py migrate (Windows)

### Option 2: Cloud PostgreSQL (e.g., Supabase, Railway, Neon)

1. Get your connection string from your provider

2. Update settings.py:

` + "```" + `python
DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.postgresql',
        'NAME': 'your_database',
        'USER': 'your_username',
        'PASSWORD': 'your_password',
        'HOST': 'your_host',
        'PORT': '5432',
    }
}
` + "```" + `

Or use an environment variable:

` + "```" + `python
import os

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.postgresql',
        'NAME': os.environ.get('DB_NAME', 'myproject'),
        'USER': os.environ.get('DB_USER', 'postgres'),
        'PASSWORD': os.environ.get('DB_PASSWORD', ''),
        'HOST': os.environ.get('DB_HOST', 'localhost'),
        'PORT': os.environ.get('DB_PORT', '5432'),
    }
}
` + "```" + `

## Common Issues

### psycopg2.errors.InvalidCatalogOrder

If you see this error, make sure PostgreSQL is running and credentials are correct.

### Connection refused

Check that PostgreSQL is running and the port (default 5432) is not blocked.

## Next Steps

1. Review your database configuration in <PROJECT_NAME>/settings.py
2. Run migrations to create database tables
3. Start the development server: ./.venv/bin/python manage.py runserver
`
}

package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (e *Env) applyNinja(projectPath string) error {
	if !e.Cfg.SetupNinja {
		return nil
	}

	if err := e.installPackages(projectPath, "Django Ninja", PkgNinja); err != nil {
		return err
	}

	settingsPath := filepath.Join(projectPath, e.Cfg.ModuleName(), SettingsFile)
	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	if !se.Contains("NINJA_PAGINATION_PER_PAGE") {
		se.AppendBlock(ninjaSettingsBlock)
		if err := se.Save(); err != nil {
			return err
		}
		e.Step("Configured Django Ninja pagination and throttle rates.")
	}

	if e.Cfg.AppName != "" {
		apiFiles := generateNinjaPackage(e.Cfg.AppName, e.Cfg.ProjectName)
		if err := e.createExampleAPI(projectPath, apiFiles); err != nil {
			return err
		}
	}

	endpointsPath := filepath.Join(projectPath, "API_ENDPOINTS.md")
	if err := os.WriteFile(endpointsPath, []byte(e.generateNinjaEndpointsMd()), 0644); err != nil {
		return fmt.Errorf("failed to create API_ENDPOINTS.md: %w", err)
	}
	e.Step("Created API_ENDPOINTS.md with API documentation URLs.")

	return nil
}

const ninjaSettingsBlock = `
NINJA_PAGINATION_CLASS = 'ninja.pagination.PageNumberPagination'

NINJA_PAGINATION_PER_PAGE = 20

NINJA_DEFAULT_THROTTLE_RATES = {
    'anon': '100/h',
    'auth': '1000/h',
    'user': '1000/h',
}
`

func (e *Env) generateNinjaEndpointsMd() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s API Endpoints\n\n", e.Cfg.ProjectName)
	sb.WriteString("## API Documentation\n\n")
	sb.WriteString("| URL | Description |\n")
	sb.WriteString("|-----|-------------|\n")
	sb.WriteString("| [http://localhost:8000/api/v1/docs](http://localhost:8000/api/v1/docs) | Swagger UI (interactive API docs) |\n")
	sb.WriteString("| [http://localhost:8000/api/v1/openapi.json](http://localhost:8000/api/v1/openapi.json) | Raw OpenAPI schema |\n")
	sb.WriteString("\n## Authentication\n\n")
	sb.WriteString("The API is **read-only for anonymous callers**. `GET` works without signing in; ")
	sb.WriteString("`POST`, `PUT` and `DELETE` use Django session authentication and answer ")
	sb.WriteString("HTTP 401 until you sign in.\n\n")
	sb.WriteString("Create a superuser (see README.md), sign in at the admin, then the write ")
	sb.WriteString("endpoints accept your session cookie.\n\n")
	sb.WriteString("To make the demo API writable without signing in, remove `auth=django_auth` ")
	sb.WriteString("from the write operations in `api/router.py`.\n")

	if e.Cfg.AppName != "" {
		fmt.Fprintf(&sb, "\n## %s API\n\n", e.Cfg.AppName)
		sb.WriteString("| Method | URL | Description |\n")
		sb.WriteString("|--------|-----|-------------|\n")
		sb.WriteString("| GET | `/api/v1/books` | List books, paginated, `?search=` filters |\n")
		sb.WriteString("| GET | `/api/v1/books/recent` | Books added recently |\n")
		sb.WriteString("| GET | `/api/v1/books/{id}` | A single book |\n")
		sb.WriteString("| POST | `/api/v1/books` | Create a book |\n")
		sb.WriteString("| PUT | `/api/v1/books/{id}` | Replace a book |\n")
		sb.WriteString("| DELETE | `/api/v1/books/{id}` | Delete a book |\n")
	}

	sb.WriteString("\n## Admin\n\n")
	sb.WriteString("| URL | Description |\n")
	sb.WriteString("|-----|-------------|\n")
	sb.WriteString("| [http://localhost:8000/admin/](http://localhost:8000/admin/) | Django admin interface |\n")

	return sb.String()
}

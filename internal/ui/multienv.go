package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SetupMultiEnvSettings reads the completed settings.py and splits it into
// a settings/ directory with base.py, local.py, production.py, and test.py.
// This MUST run after all other settings-modifying steps.
func (m *Model) SetupMultiEnvSettings(projectPath string) error {
	m.UpdateProgress("Splitting settings into multi-env layout...")

	moduleName := m.ProjectModuleName()
	modulePath := filepath.Join(projectPath, moduleName)
	settingsPath := filepath.Join(modulePath, SettingsFile)

	// Read the completed settings.py
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("failed to read settings.py for multi-env split: %w", err)
	}
	content := string(data)

	// Create settings directory
	settingsDir := filepath.Join(modulePath, "settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	// Generate all settings files
	basePy := m.generateBasePy(content)
	localPy := m.generateLocalPy()
	productionPy := m.generateProductionPy()
	testPy := m.generateTestPy()
	initPy := `# Settings split into base, local, production, and test.
# Default to local settings for development.
`

	// Write settings files
	files := map[string]string{
		"__init__.py":   initPy,
		"base.py":       basePy,
		"local.py":      localPy,
		"production.py": productionPy,
		"test.py":       testPy,
	}

	for name, fileContent := range files {
		path := filepath.Join(settingsDir, name)
		if err := os.WriteFile(path, []byte(fileContent), 0644); err != nil {
			return fmt.Errorf("failed to write settings/%s: %w", name, err)
		}
	}

	// Delete original settings.py
	if err := os.Remove(settingsPath); err != nil {
		return fmt.Errorf("failed to remove original settings.py: %w", err)
	}

	// Update manage.py
	if err := m.updateSettingsModule(filepath.Join(projectPath, ManageFile), moduleName+".settings.local"); err != nil {
		return err
	}

	// Update wsgi.py
	if err := m.updateSettingsModule(filepath.Join(modulePath, "wsgi.py"), moduleName+".settings.production"); err != nil {
		return err
	}

	// Update asgi.py
	if err := m.updateSettingsModule(filepath.Join(modulePath, "asgi.py"), moduleName+".settings.production"); err != nil {
		return err
	}

	// Add DJANGO_SETTINGS_MODULE to .env
	envPath := filepath.Join(projectPath, EnvFile)
	if envData, err := os.ReadFile(envPath); err == nil {
		envContent := string(envData)
		if !strings.Contains(envContent, "DJANGO_SETTINGS_MODULE") {
			envContent += fmt.Sprintf("\n# === Settings Module ===\nDJANGO_SETTINGS_MODULE=%s.settings.local\n", moduleName)
			os.WriteFile(envPath, []byte(envContent), 0600)
		}
	}

	// Generate .env.production.example and .env.staging.example
	if err := m.generateEnvExampleFiles(projectPath, moduleName); err != nil {
		return err
	}

	m.AppendStepMessage("Settings split into settings/base.py, local.py, production.py, test.py")
	m.AppendStepMessage("manage.py → settings.local (development)")
	m.AppendStepMessage("wsgi.py/asgi.py → settings.production")
	m.AppendStepMessage("Created: .env.production.example, .env.staging.example")
	return nil
}

// generateBasePy creates base.py from the original settings.py content.
// It removes DEBUG, ALLOWED_HOSTS assignments and the `if not DEBUG:` security block
// (those move to local.py and production.py respectively).
func (m *Model) generateBasePy(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	skip := false

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		// Skip DEBUG assignment (it goes to local/production)
		if strings.HasPrefix(trimmed, "DEBUG") && strings.Contains(trimmed, "=") && !strings.Contains(trimmed, "SECURE") {
			continue
		}

		// Skip ALLOWED_HOSTS assignment (it goes to local/production)
		if strings.HasPrefix(trimmed, "ALLOWED_HOSTS") && strings.Contains(trimmed, "=") {
			continue
		}

		// Skip the `if not DEBUG:` block (it goes to production.py)
		if trimmed == "if not DEBUG:" {
			skip = true
			// Also skip preceding blank lines
			for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
				result = result[:len(result)-1]
			}
			continue
		}
		if skip {
			if trimmed == "" || strings.HasPrefix(lines[i], "    ") || strings.HasPrefix(lines[i], "\t") {
				continue
			}
			skip = false
		}

		result = append(result, lines[i])
	}

	base := strings.Join(result, "\n")

	// Collapse runs of 3+ blank lines into 2
	for strings.Contains(base, "\n\n\n") {
		base = strings.ReplaceAll(base, "\n\n\n", "\n\n")
	}

	// Adjust BASE_DIR — one more .parent since settings is now in a subdirectory
	base = strings.Replace(base,
		"BASE_DIR = Path(__file__).resolve().parent.parent",
		"BASE_DIR = Path(__file__).resolve().parent.parent.parent",
		1)

	return base
}

// generateLocalPy creates the local (development) settings.
func (m *Model) generateLocalPy() string {
	return `from .base import *  # noqa: F401,F403

DEBUG = True

ALLOWED_HOSTS = ["localhost", "127.0.0.1"]

# Django Debug Toolbar (optional — install django-debug-toolbar)
# INSTALLED_APPS += ["debug_toolbar"]
# MIDDLEWARE.insert(0, "debug_toolbar.middleware.DebugToolbarMiddleware")
# INTERNAL_IPS = ["127.0.0.1"]

# Email backend for development
EMAIL_BACKEND = "django.core.mail.backends.console.EmailBackend"
`
}

// generateProductionPy creates the production settings.
func (m *Model) generateProductionPy() string {
	return `from .base import *  # noqa: F401,F403

DEBUG = False

ALLOWED_HOSTS = env.list("ALLOWED_HOSTS")

# Security settings
SECURE_SSL_REDIRECT = env.bool("SECURE_SSL_REDIRECT", default=True)
SECURE_HSTS_SECONDS = env.int("SECURE_HSTS_SECONDS", default=31536000)
SECURE_HSTS_INCLUDE_SUBDOMAINS = True
SECURE_HSTS_PRELOAD = True
SECURE_PROXY_SSL_HEADER = ("HTTP_X_FORWARDED_PROTO", "https")
SESSION_COOKIE_SECURE = True
CSRF_COOKIE_SECURE = True
`
}

// generateTestPy creates the test settings.
func (m *Model) generateTestPy() string {
	return `from .base import *  # noqa: F401,F403

DEBUG = False

DATABASES = {
    "default": {
        "ENGINE": "django.db.backends.sqlite3",
        "NAME": BASE_DIR / "test_db.sqlite3",
    }
}

PASSWORD_HASHERS = ["django.contrib.auth.hashers.MD5PasswordHasher"]
EMAIL_BACKEND = "django.core.mail.backends.locmem.EmailBackend"
`
}

// updateSettingsModule rewrites the DJANGO_SETTINGS_MODULE default in a file.
func (m *Model) updateSettingsModule(filePath, newModule string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filepath.Base(filePath), err)
	}

	content := string(data)
	moduleName := m.ProjectModuleName()

	// Replace the settings module reference
	oldSetting := fmt.Sprintf("%s.settings", moduleName)
	content = strings.Replace(content, oldSetting, newModule, 1)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filepath.Base(filePath), err)
	}
	return nil
}

// generateEnvExampleFiles creates .env.production.example and .env.staging.example.
func (m *Model) generateEnvExampleFiles(projectPath, moduleName string) error {
	dbVars := ""
	switch m.DatabaseType {
	case "postgres-docker", "postgres-string":
		dbVars = `
# Database (PostgreSQL)
DB_NAME=your_db_name
DB_USER=postgres
DB_PASSWORD=<strong-password-here>
DB_HOST=localhost
DB_PORT=5432
`
	case "mysql-docker":
		dbVars = `
# Database (MySQL)
DB_NAME=your_db_name
DB_USER=django_user
DB_PASSWORD=<strong-password-here>
DB_HOST=localhost
DB_PORT=3306
`
	default:
		dbVars = `
# Database (SQLite — consider PostgreSQL for production)
# No database environment variables needed for SQLite.
`
	}

	prodExample := fmt.Sprintf(`# Production Environment Variables
# Copy to .env on your production server and fill in the values.

SECRET_KEY=<generate-a-strong-secret-key>
DEBUG=False
ALLOWED_HOSTS=yourdomain.com,www.yourdomain.com
DJANGO_SETTINGS_MODULE=%s.settings.production

# Security
SECURE_SSL_REDIRECT=True
SECURE_HSTS_SECONDS=31536000
%s`, moduleName, dbVars)

	stagingExample := fmt.Sprintf(`# Staging Environment Variables
# Copy to .env on your staging server and fill in the values.

SECRET_KEY=<generate-a-strong-secret-key>
DEBUG=False
ALLOWED_HOSTS=staging.yourdomain.com
DJANGO_SETTINGS_MODULE=%s.settings.production

# Security
SECURE_SSL_REDIRECT=True
SECURE_HSTS_SECONDS=0
%s`, moduleName, dbVars)

	envFiles := map[string]string{
		".env.production.example": prodExample,
		".env.staging.example":   stagingExample,
	}

	for name, content := range envFiles {
		path := filepath.Join(projectPath, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}
	return nil
}

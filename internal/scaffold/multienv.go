package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// applyMultiEnvSettings reads the completed settings.py and splits it into
// a settings/ directory with base.py, local.py, production.py, and test.py.
// This MUST run after all other settings-modifying steps.
func (e *Env) applyMultiEnvSettings(projectPath string) error {
	moduleName := e.Cfg.ModuleName()
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

	// base.py is derived from the project's own settings.py; the other three are
	// fixed content and live in templates/settings/.
	files := map[string]string{
		"__init__.py":   renderStatic("init.py.tmpl"),
		"base.py":       e.generateBasePy(content),
		"local.py":      renderStatic("local.py.tmpl"),
		"production.py": renderStatic("production.py.tmpl"),
		"test.py":       renderStatic("test.py.tmpl"),
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
	if err := e.updateSettingsModule(filepath.Join(projectPath, ManageFile), moduleName+".settings.local"); err != nil {
		return err
	}

	// Update wsgi.py
	if err := e.updateSettingsModule(filepath.Join(modulePath, "wsgi.py"), moduleName+".settings.production"); err != nil {
		return err
	}

	// Update asgi.py
	if err := e.updateSettingsModule(filepath.Join(modulePath, "asgi.py"), moduleName+".settings.production"); err != nil {
		return err
	}

	// Add DJANGO_SETTINGS_MODULE to .env. Without it the generated project has
	// no settings module to load once settings.py becomes a package, so a
	// failure here must abort rather than report success.
	envPath := filepath.Join(projectPath, EnvFile)
	if envData, err := os.ReadFile(envPath); err == nil {
		envContent := string(envData)
		if !strings.Contains(envContent, "DJANGO_SETTINGS_MODULE") {
			envContent += fmt.Sprintf("\n# === Settings Module ===\nDJANGO_SETTINGS_MODULE=%s.settings.local\n", moduleName)
			if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
				return fmt.Errorf("failed to record DJANGO_SETTINGS_MODULE in %s: %w", EnvFile, err)
			}
		}
	}

	// Generate .env.production.example and .env.staging.example
	if err := e.generateEnvExampleFiles(projectPath, moduleName); err != nil {
		return err
	}

	e.Step("Settings split into settings/base.py, local.py, production.py, test.py")
	e.Step("manage.py → settings.local (development)")
	e.Step("wsgi.py/asgi.py → settings.production")
	e.Step("Created: .env.production.example, .env.staging.example")
	return nil
}

// generateBasePy creates base.py from the original settings.py content.
// It removes DEBUG, ALLOWED_HOSTS assignments and the `if not DEBUG:` security block
// (those move to local.py and production.py respectively).
func (e *Env) generateBasePy(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	skip := false

	for i := range lines {
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

// updateSettingsModule rewrites the DJANGO_SETTINGS_MODULE default in a file.
func (e *Env) updateSettingsModule(filePath, newModule string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filepath.Base(filePath), err)
	}

	content := string(data)
	moduleName := e.Cfg.ModuleName()

	// Replace the settings module reference
	oldSetting := fmt.Sprintf("%s.settings", moduleName)
	content = strings.Replace(content, oldSetting, newModule, 1)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filepath.Base(filePath), err)
	}
	return nil
}

// generateEnvExampleFiles creates .env.production.example and .env.staging.example.
func (e *Env) generateEnvExampleFiles(projectPath, moduleName string) error {
	dbVars := ""
	switch e.Cfg.DatabaseType {
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
		".env.staging.example":    stagingExample,
	}

	for name, content := range envFiles {
		path := filepath.Join(projectPath, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}
	return nil
}

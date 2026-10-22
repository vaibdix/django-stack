package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
)

type multiEnvSettingsData struct {
	ModuleName string
	Templates  bool
	AllAuth    bool
	Mailers    bool
	NativeCSP  bool
	Env        productionEnvExpressions

	FromEmailGuard string
}

func (multiEnvSettingsData) PlaceholderSecretKey() string { return placeholderSecretKey }

func (e *Env) applyMultiEnvSettings(projectPath string) error {
	moduleName := e.Cfg.ModuleName()
	modulePath := filepath.Join(projectPath, moduleName)
	settingsPath := filepath.Join(modulePath, SettingsFile)

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("failed to read settings.py for multi-env split: %w", err)
	}
	content := string(data)

	settingsDir := filepath.Join(modulePath, "settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	base, err := e.generateBasePy(content)
	if err != nil {
		return err
	}

	settingsData := multiEnvSettingsData{
		ModuleName: moduleName,
		Templates:  e.Cfg.CreateTemplates,
		AllAuth:    e.Cfg.SetupAllAuth,
		Mailers:    config.DjangoSupportsMailers(e.Cfg.DjangoVersion),
		NativeCSP:  config.DjangoSupportsNativeCSP(e.Cfg.DjangoVersion),
		Env:        e.productionEnvExpressions(),

		FromEmailGuard: e.fromEmailGuard(""),
	}

	files := map[string]string{
		"__init__.py":   "",
		"base.py":       base,
		"local.py":      render("local.py.tmpl", settingsData),
		"production.py": render("production.py.tmpl", settingsData),
		"test.py":       render("test.py.tmpl", settingsData),
	}

	for name, fileContent := range files {
		path := filepath.Join(settingsDir, name)
		if err := os.WriteFile(path, []byte(fileContent), 0644); err != nil {
			return fmt.Errorf("failed to write settings/%s: %w", name, err)
		}
	}

	if err := os.Remove(settingsPath); err != nil {
		return fmt.Errorf("failed to remove original settings.py: %w", err)
	}

	if err := e.updateSettingsModule(filepath.Join(projectPath, ManageFile), moduleName+".settings.local"); err != nil {
		return err
	}

	if err := e.updateSettingsModule(filepath.Join(modulePath, "wsgi.py"), moduleName+".settings.production"); err != nil {
		return err
	}

	if err := e.updateSettingsModule(filepath.Join(modulePath, "asgi.py"), moduleName+".settings.production"); err != nil {
		return err
	}

	if err := e.generateEnvExampleFiles(projectPath, moduleName); err != nil {
		return err
	}

	e.Step("Settings split into settings/base.py, local.py, production.py, test.py")
	e.Step("manage.py → settings.local (development)")
	e.Step("wsgi.py/asgi.py → settings.production")
	e.Step("Created: .env.production.example, .env.staging.example")
	return nil
}

func (e *Env) generateBasePy(content string) (string, error) {

	se := &SettingsEditor{content: content}
	if err := se.RemoveCoreSetting("ALLOWED_HOSTS"); err != nil {
		return "", err
	}

	se.StripSettingBlock("MAILERS")
	base := se.Content()

	for strings.Contains(base, "\n\n\n") {
		base = strings.ReplaceAll(base, "\n\n\n", "\n\n")
	}

	return e.rebaseBaseDir(base)
}

func (e *Env) rebaseBaseDir(base string) (string, error) {
	if e.Cfg.SetupPydanticConf {
		return base, nil
	}
	const single = "BASE_DIR = Path(__file__).resolve().parent.parent"
	if !strings.Contains(base, single) {
		return "", &AnchorError{Edit: "BASE_DIR for settings/base.py", Anchor: single}
	}
	return strings.Replace(base, single, single+".parent", 1), nil
}

func stripSettingBlock(content, name string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, name+" = {") {
			result = append(result, lines[i])
			continue
		}

		for len(result) > 0 {
			prev := strings.TrimSpace(result[len(result)-1])
			if prev == "" || strings.HasPrefix(prev, "#") {
				result = result[:len(result)-1]
				continue
			}
			break
		}

		depth := 0
		for i < len(lines) {
			depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
			i++
			if depth <= 0 {
				break
			}
		}
		i--
	}

	return strings.Join(result, "\n")
}

func (e *Env) updateSettingsModule(filePath, newModule string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filepath.Base(filePath), err)
	}

	content := string(data)
	moduleName := e.Cfg.ModuleName()

	oldSetting := regexp.MustCompile(`(['"])` + regexp.QuoteMeta(moduleName+".settings") + `(['"])`)
	at := oldSetting.FindStringSubmatchIndex(content)
	if at == nil {
		return &AnchorError{Edit: "the settings module in " + filepath.Base(filePath), Anchor: "'" + moduleName + ".settings'"}
	}
	quote := content[at[2]:at[3]]
	content = content[:at[0]] + quote + newModule + quote + content[at[1]:]

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filepath.Base(filePath), err)
	}
	return nil
}

func (e *Env) generateEnvExampleFiles(projectPath, moduleName string) error {
	var dbVars string
	switch e.Cfg.DatabaseType {
	case config.PostgresDocker:
		dbVars = `
# Database (PostgreSQL)
DB_NAME=your_db_name
DB_USER=django_user
DB_PASSWORD=<strong-password-here>
DB_HOST=localhost
DB_PORT=5432
`
	case config.MySQLDocker:
		dbVars = `
# Database (MySQL)
DB_NAME=your_db_name
DB_USER=django_user
DB_PASSWORD=<strong-password-here>
DB_HOST=127.0.0.1
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

# Generate with: python -c 'import secrets; print(secrets.token_urlsafe(50))'
SECRET_KEY=<paste-a-50-character-key-here>
ENVIRONMENT=production
ALLOWED_HOSTS=yourdomain.com,www.yourdomain.com
CSRF_TRUSTED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
DEFAULT_FROM_EMAIL=no-reply@yourdomain.com
# DJANGO_SETTINGS_MODULE is read before this file, so setting it here has no effect.
# Export it in the server's environment instead:
#   export DJANGO_SETTINGS_MODULE=%s.settings.production

# Security
SECURE_SSL_REDIRECT=True
SECURE_HSTS_SECONDS=31536000
# Turn these on only once every subdomain serves HTTPS; preload is hard to undo.
SECURE_HSTS_INCLUDE_SUBDOMAINS=False
SECURE_HSTS_PRELOAD=False
# Only enable behind a proxy that strips a client-supplied X-Forwarded-Proto.
TRUST_PROXY_SSL_HEADER=False
%s%s`, moduleName, dbVars, e.deployedSentrySection("production"))

	stagingExample := fmt.Sprintf(`# Staging Environment Variables
# Copy to .env on your staging server and fill in the values.

# Generate with: python -c 'import secrets; print(secrets.token_urlsafe(50))'
SECRET_KEY=<paste-a-50-character-key-here>
ENVIRONMENT=staging
ALLOWED_HOSTS=staging.yourdomain.com
CSRF_TRUSTED_ORIGINS=https://staging.yourdomain.com
DEFAULT_FROM_EMAIL=no-reply@staging.yourdomain.com
# DJANGO_SETTINGS_MODULE is read before this file, so setting it here has no effect.
# Export it in the server's environment instead:
#   export DJANGO_SETTINGS_MODULE=%s.settings.production

# Security
SECURE_SSL_REDIRECT=True
SECURE_HSTS_SECONDS=0
SECURE_HSTS_INCLUDE_SUBDOMAINS=False
SECURE_HSTS_PRELOAD=False
# Only enable behind a proxy that strips a client-supplied X-Forwarded-Proto.
TRUST_PROXY_SSL_HEADER=False
%s%s`, moduleName, dbVars, e.deployedSentrySection("staging"))

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

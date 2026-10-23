package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) configPackages() []string {
	if e.cfg.Has(config.Pydantic) {
		return []string{pkgPydantic, pkgPydanticSettings}
	}
	return []string{pkgEnviron}
}

func (e *Env) settingsEditor(projectPath string) (*settingsEditor, error) {
	return newSettingsEditor(filepath.Join(projectPath, e.cfg.ModuleName(), settingsFile))
}

func (e *Env) envConfig() envConfig {
	return envConfig{
		Database:   e.cfg.DatabaseType,
		Pydantic:   e.cfg.Has(config.Pydantic),
		ModuleName: e.cfg.ModuleName(),
	}
}

type confTemplateData struct {
	Database bool
	DBUser   string
	DBHost   string
	DBPort   string
	Sentry   bool
	Ruff     bool
}

func (confTemplateData) TestSecretKey() string { return testSecretKey }

func (e *Env) confTemplateData() confTemplateData {
	data := confTemplateData{Sentry: e.cfg.Has(config.Sentry), Ruff: e.cfg.Has(config.Ruff)}

	if e.cfg.DatabaseType.UsesDocker() {
		database := e.cfg.DatabaseType.Spec()
		data.Database, data.DBUser, data.DBHost, data.DBPort = true, "django_user", database.Host, strconv.Itoa(database.Port)
	}

	return data
}

func (e *Env) writeConfModule(projectPath string) error {
	if !e.cfg.Has(config.Pydantic) {
		return nil
	}

	path := filepath.Join(projectPath, e.cfg.ModuleName(), confFile)
	if err := os.WriteFile(path, []byte(render("conf.py.tmpl", e.confTemplateData())), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", confFile, err)
	}

	e.Step("Created: " + e.cfg.ModuleName() + "/" + confFile + " (typed settings validated at startup)")
	return nil
}

type envReader struct{ pydantic bool }

func (e *Env) env() envReader { return envReader{pydantic: e.cfg.Has(config.Pydantic)} }

func (r envReader) required(name string) string {
	if r.pydantic {
		return "env." + name
	}
	return fmt.Sprintf("env('%s')", name)
}

func (r envReader) requiredSecret(name string) string {
	if r.pydantic {
		return "env." + name + ".get_secret_value()"
	}
	return fmt.Sprintf("env('%s')", name)
}

func (r envReader) secret(name string) string {
	if r.pydantic {
		return "env." + name + ".get_secret_value()"
	}
	return fmt.Sprintf("env('%s', default='')", name)
}

func (r envReader) adminURL() string {
	if r.pydantic {
		return "env.ADMIN_URL"
	}
	return "env('ADMIN_URL', default='admin/')"
}

func (r envReader) debug() string {
	if r.pydantic {
		return "env.DEBUG"
	}
	return "ENVIRONMENT in {'local', 'test'}"
}

const testSecretKey = "insecure-test-only-key-used-when-environment-is-test-and-nowhere-else"

func (r envReader) secretKey() string {
	if r.pydantic {
		return "env.SECRET_KEY.get_secret_value()"
	}
	return "env('SECRET_KEY', default='" + testSecretKey + "') if ENVIRONMENT == 'test' else env('SECRET_KEY')"
}

func (r envReader) environment() string {
	if r.pydantic {
		return "env.ENVIRONMENT"
	}
	return "env('ENVIRONMENT', default='production')"
}

func (r envReader) str(name, fallback string) string {
	if r.pydantic {
		return "env." + name
	}
	return fmt.Sprintf("env('%s', default='%s')", name, fallback)
}

func (r envReader) boolean(name string, fallback bool) string {
	if r.pydantic {
		return "env." + name
	}
	return fmt.Sprintf("env_bool('%s', default=%s)", name, pythonBool(fallback))
}

func (r envReader) integer(name string, fallback int) string {
	if r.pydantic {
		return "env." + name
	}
	return fmt.Sprintf("env.int('%s', default=%d)", name, fallback)
}

func (r envReader) listRequired(name string) string {
	if r.pydantic {
		return "env." + name
	}
	return fmt.Sprintf("env.list('%s')", name)
}

func (r envReader) list(name string, fallback ...string) string {
	if r.pydantic {
		return "env." + name
	}

	quoted := make([]string, 0, len(fallback))
	for _, item := range fallback {
		quoted = append(quoted, "'"+item+"'")
	}
	return fmt.Sprintf("env.list('%s', default=[%s])", name, strings.Join(quoted, ", "))
}

func (r envReader) number(name string, fallback float64) string {
	if r.pydantic {
		return "env." + name
	}
	return fmt.Sprintf("env.float('%s', default=%s)", name, strconv.FormatFloat(fallback, 'f', -1, 64))
}

type productionEnvExpressions struct {
	SecretKey        string
	AllowedHosts     string
	SSLRedirect      string
	HSTSSeconds      string
	HSTSSubdomains   string
	HSTSPreload      string
	TrustProxyHeader string
	CSRFOrigins      string
	EmailHost        string
	EmailPort        string
	EmailUser        string
	EmailPassword    string
	EmailUseTLS      string
	DefaultFromEmail string
}

func (e *Env) productionEnvExpressions() productionEnvExpressions {
	r := e.env()
	return productionEnvExpressions{
		SecretKey:        r.requiredSecret("SECRET_KEY"),
		AllowedHosts:     r.listRequired("ALLOWED_HOSTS"),
		SSLRedirect:      r.boolean("SECURE_SSL_REDIRECT", true),
		HSTSSeconds:      r.integer("SECURE_HSTS_SECONDS", 31536000),
		HSTSSubdomains:   r.boolean("SECURE_HSTS_INCLUDE_SUBDOMAINS", false),
		HSTSPreload:      r.boolean("SECURE_HSTS_PRELOAD", false),
		TrustProxyHeader: r.boolean("TRUST_PROXY_SSL_HEADER", false),
		CSRFOrigins:      r.list("CSRF_TRUSTED_ORIGINS"),
		EmailHost:        r.str("EMAIL_HOST", "localhost"),
		EmailPort:        r.integer("EMAIL_PORT", 587),
		EmailUser:        r.str("EMAIL_HOST_USER", ""),
		EmailPassword:    r.secret("EMAIL_HOST_PASSWORD"),
		EmailUseTLS:      r.boolean("EMAIL_USE_TLS", true),
		DefaultFromEmail: r.str("DEFAULT_FROM_EMAIL", djangoDefaultFromEmail),
	}
}

func pythonBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

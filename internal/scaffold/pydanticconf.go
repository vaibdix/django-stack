package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
)

// configPackages names the library that reads .env. The two are alternatives,
// never both: a settings module with two config readers in it is worse than
// either one on its own.
func (e *Env) configPackages() []string {
	if e.Cfg.SetupPydanticConf {
		return []string{PkgPydantic, PkgPydanticSettings}
	}
	return []string{PkgEnviron}
}

// settingsEditor opens settings.py already knowing which package is first-party,
// so any import it adds lands in the right isort section.
func (e *Env) settingsEditor(projectPath string) (*SettingsEditor, error) {
	se, err := NewSettingsEditor(filepath.Join(projectPath, e.Cfg.ModuleName(), SettingsFile))
	if err != nil {
		return nil, err
	}
	se.firstParty = e.Cfg.ModuleName()
	return se, nil
}

func (e *Env) envConfig() EnvConfig {
	return EnvConfig{
		Database:   e.Cfg.DatabaseType,
		Pydantic:   e.Cfg.SetupPydanticConf,
		ModuleName: e.Cfg.ModuleName(),
	}
}

type confTemplateData struct {
	Database bool
	DBUser   string
	DBPort   string
	Sentry   bool
}

func (e *Env) confTemplateData() confTemplateData {
	data := confTemplateData{Sentry: e.Cfg.SetupSentry}

	switch e.Cfg.DatabaseType {
	case config.PostgresDocker, config.PostgresString:
		data.Database, data.DBUser, data.DBPort = true, "postgres", defaultPostgresPort
	case config.MySQLDocker:
		data.Database, data.DBUser, data.DBPort = true, "django_user", defaultMySQLPort
	}

	return data
}

// writeConfModule generates <module>/conf.py. It sits beside settings.py rather
// than inside it so that the multi-env split, which turns settings.py into a
// package, never moves it and never invalidates the import path.
func (e *Env) writeConfModule(projectPath string) error {
	if !e.Cfg.SetupPydanticConf {
		return nil
	}

	path := filepath.Join(projectPath, e.Cfg.ModuleName(), ConfFile)
	if err := os.WriteFile(path, []byte(render("conf.py.tmpl", e.confTemplateData())), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", ConfFile, err)
	}

	e.Step("Created: " + e.Cfg.ModuleName() + "/" + ConfFile + " (typed settings validated at startup)")
	return nil
}

// envReader renders the expression that reads one environment variable. Both
// backends land on a single name here, so a generator never has to know which
// library the project ended up with.
type envReader struct{ pydantic bool }

func (e *Env) env() envReader { return envReader{pydantic: e.Cfg.SetupPydanticConf} }

func (r envReader) required(name string) string {
	if r.pydantic {
		return "env." + name
	}
	return fmt.Sprintf("env('%s')", name)
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
	return fmt.Sprintf("env.bool('%s', default=%s)", name, pythonBool(fallback))
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

// productionEnvExpressions feeds production.py.tmpl, which reads more of the
// environment than any other generated settings module.
type productionEnvExpressions struct {
	SecretKey     string
	AllowedHosts  string
	SSLRedirect   string
	HSTSSeconds   string
	EmailHost     string
	EmailPort     string
	EmailUser     string
	EmailPassword string
	EmailUseTLS   string
}

func (e *Env) productionEnvExpressions() productionEnvExpressions {
	r := e.env()
	return productionEnvExpressions{
		SecretKey:     r.required("SECRET_KEY"),
		AllowedHosts:  r.listRequired("ALLOWED_HOSTS"),
		SSLRedirect:   r.boolean("SECURE_SSL_REDIRECT", true),
		HSTSSeconds:   r.integer("SECURE_HSTS_SECONDS", 31536000),
		EmailHost:     r.str("EMAIL_HOST", "localhost"),
		EmailPort:     r.integer("EMAIL_PORT", 587),
		EmailUser:     r.str("EMAIL_HOST_USER", ""),
		EmailPassword: r.str("EMAIL_HOST_PASSWORD", ""),
		EmailUseTLS:   r.boolean("EMAIL_USE_TLS", true),
	}
}

func pythonBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

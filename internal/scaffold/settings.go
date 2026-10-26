package scaffold

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

type envConfig struct {
	Database config.Database

	Pydantic bool

	ModuleName string
}

func (c envConfig) reader() envReader { return envReader{pydantic: c.Pydantic} }

func (c envConfig) importLine() string {
	if c.Pydantic {
		return "from " + c.ModuleName + "." + strings.TrimSuffix(confFile, ".py") + " import env"
	}
	return "import environ"
}

func (c envConfig) preamble() string {
	if c.Pydantic {
		return ""
	}
	return `env = environ.Env()
environ.Env.read_env(BASE_DIR / ".env")


` + envBoolFunction() + `
`
}

func envBoolFunction() string { return renderStatic("env_bool.py.tmpl") }

func envConfigBlock(cfg envConfig) string {
	r := cfg.reader()

	database := cfg.Database.Spec()
	databaseBlock := `DATABASES = {
    "default": {
        "ENGINE": "` + database.Engine + `",
        "NAME": BASE_DIR / "db.sqlite3",
    }
}`
	if cfg.Database.UsesDocker() {
		var options strings.Builder
		if len(database.Options) > 0 {
			options.WriteString("        \"OPTIONS\": {\n")
			for _, option := range database.Options {
				options.WriteString("            " + option + ",\n")
			}
			options.WriteString("        },\n")
		}
		databaseBlock = fmt.Sprintf(`DATABASES = {
    "default": {
        "ENGINE": "%s",
        "NAME": %s,
        "USER": %s,
        "PASSWORD": %s,
        "HOST": %s,
        "PORT": %s,
%s        "CONN_MAX_AGE": %s,
        "CONN_HEALTH_CHECKS": True,
    }
}`, database.Engine, r.required("DB_NAME"), r.str("DB_USER", "django_user"), r.secret("DB_PASSWORD"),
			r.str("DB_HOST", database.Host), r.str("DB_PORT", strconv.Itoa(database.Port)), options.String(),
			r.integer("DB_CONN_MAX_AGE", 60))
	}

	return strings.TrimLeft(cfg.preamble()+`
ENVIRONMENT = `+r.environment()+`

ADMIN_URL = `+r.adminURL()+`

SECRET_KEY = `+r.secretKey()+`

DEBUG = `+r.debug()+`

ALLOWED_HOSTS = `+r.list("ALLOWED_HOSTS", "localhost", "127.0.0.1")+`

`+databaseBlock, "\n")
}

func enforceCSP(nativeCSP bool, indent string) string {
	enforced, reported := "CONTENT_SECURITY_POLICY", "CONTENT_SECURITY_POLICY_REPORT_ONLY"
	if nativeCSP {
		enforced, reported = "SECURE_CSP", "SECURE_CSP_REPORT_ONLY"
	}
	return indent + enforced + " = " + reported + "\n" +
		indent + reported + " = {}\n"
}

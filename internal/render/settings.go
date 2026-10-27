package render

import (
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

type envConfig struct {
	Database config.Database

	Pydantic bool

	ModuleName string

	SettingsPackage bool
}

func (c envConfig) reader() envReader { return envReader{pydantic: c.Pydantic} }

func (c envConfig) importLine() string {
	if c.Pydantic {
		return "from " + c.ModuleName + "." + strings.TrimSuffix(ConfFile, ".py") + " import env"
	}
	return "import environ"
}

func envBoolFunction() string { return RenderStatic("env_bool.py.tmpl") }

type envBlockData struct {
	Environ         bool
	EnvBool         string
	Environment     string
	AdminURL        string
	SecretKey       string
	Debug           string
	SettingsPackage bool
	AllowedHosts    string
	Engine          string
	Docker          bool
	Name            string
	User            string
	Password        string
	Host            string
	Port            string
	Options         []string
	ConnMaxAge      string
}

func envConfigBlock(cfg envConfig) string {
	r := cfg.reader()
	database := cfg.Database.Spec()
	data := envBlockData{
		Environ:         !cfg.Pydantic,
		Environment:     r.environment(),
		AdminURL:        r.adminURL(),
		SecretKey:       r.secretKey(),
		Debug:           r.debug(),
		SettingsPackage: cfg.SettingsPackage,
		AllowedHosts:    r.list("ALLOWED_HOSTS", "localhost", "127.0.0.1"),
		Engine:          database.Engine,
		Docker:          cfg.Database.UsesDocker(),
	}
	if data.Environ {
		data.EnvBool = envBoolFunction()
	}
	if data.Docker {
		data.Name, data.User, data.Password = r.required("DB_NAME"), r.str("DB_USER", "django_user"), r.secret("DB_PASSWORD")
		data.Host, data.Port = r.str("DB_HOST", database.Host), r.str("DB_PORT", strconv.Itoa(database.Port))
		data.Options, data.ConnMaxAge = database.Options, r.integer("DB_CONN_MAX_AGE", 60)
	}
	return Render("env_config.py.tmpl", data)
}

package scaffold

import (
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
)

type multiEnvSettingsData struct {
	ModuleName string
	Templates  bool
	AllAuth    bool
	Mailers    bool
	NativeCSP  bool
	Env        productionEnvExpressions

	FromEmailGuard string
	Ruff           bool
}

func (multiEnvSettingsData) PlaceholderSecretKey() string { return placeholderSecretKey }

func (p project) multiEnvSettingsData() multiEnvSettingsData {
	return multiEnvSettingsData{
		ModuleName: p.cfg.ModuleName(),
		Templates:  p.cfg.CreateTemplates(),
		AllAuth:    p.cfg.Has(config.AllAuth),
		Mailers:    config.DjangoSupportsMailers(p.cfg.Django),
		NativeCSP:  config.DjangoSupportsNativeCSP(p.cfg.Django),
		Env:        p.productionEnvExpressions(),

		FromEmailGuard: strings.TrimSuffix(p.fromEmailGuard(""), "\n"),
		Ruff:           p.cfg.Has(config.Ruff),
	}
}

type deployedEnvironment struct {
	name        string
	hosts       string
	origins     string
	fromEmail   string
	hstsSeconds string
	hstsNote    string
}

var deployedEnvironments = []deployedEnvironment{
	{
		name: "production", hosts: "yourdomain.com,www.yourdomain.com",
		origins:   "https://yourdomain.com,https://www.yourdomain.com",
		fromEmail: "no-reply@yourdomain.com", hstsSeconds: "31536000",
		hstsNote: "# Turn these on only once every subdomain serves HTTPS; preload is hard to undo.\n",
	},
	{
		name: "staging", hosts: "staging.yourdomain.com",
		origins:   "https://staging.yourdomain.com",
		fromEmail: "no-reply@staging.yourdomain.com", hstsSeconds: "0",
	},
}

func (p project) deployedEnvExample(env deployedEnvironment, moduleName string) string {
	database := "# SQLite needs no database variables; consider PostgreSQL for production.\n"
	if vars := p.deployedDatabase().render(); vars != "" {
		database = vars + "\n"
	}
	return "# " + titleCase(env.name) + " environment variables\n" +
		"# Copy to .env on your " + env.name + " server and fill in the values.\n\n" +
		"# Generate with: python -c 'import secrets; print(secrets.token_urlsafe(50))'\n" +
		"SECRET_KEY=<paste-a-50-character-key-here>\n" +
		"ENVIRONMENT=" + env.name + "\n" +
		"ALLOWED_HOSTS=" + env.hosts + "\n" +
		"CSRF_TRUSTED_ORIGINS=" + env.origins + "\n" +
		"DEFAULT_FROM_EMAIL=" + env.fromEmail + "\n" +
		"# DJANGO_SETTINGS_MODULE is read before this file, so setting it here has no effect.\n" +
		"# Export it in the server's environment instead:\n" +
		"#   export DJANGO_SETTINGS_MODULE=" + moduleName + ".settings.production\n\n" +
		"SECURE_SSL_REDIRECT=True\n" +
		"SECURE_HSTS_SECONDS=" + env.hstsSeconds + "\n" +
		env.hstsNote +
		"SECURE_HSTS_INCLUDE_SUBDOMAINS=False\n" +
		"SECURE_HSTS_PRELOAD=False\n" +
		"# Only enable behind a proxy that strips a client-supplied X-Forwarded-Proto.\n" +
		"TRUST_PROXY_SSL_HEADER=False\n\n" +
		database + p.deployedSentrySection(env.name)
}

func (p project) deployedDatabase() dotenvDatabase {
	if !p.cfg.Database.UsesDocker() {
		return dotenvDatabase{}
	}
	database := p.cfg.Database.Spec()
	host, port := database.Host, strconv.Itoa(database.Port)
	return dotenvDatabase{Vars: []dotenvVar{
		{"DB_NAME", dbdocker.DatabaseName(p.cfg.Name)},
		{"DB_USER", "django_user"},
		{"DB_PASSWORD", "<set-password>"},
		{"DB_HOST", host},
		{"DB_PORT", port},
	}}
}

func (p project) deployedEnvFiles() fileSet {
	files := fileSet{}
	if !p.cfg.Has(config.MultiEnv) {
		return files
	}
	for _, env := range deployedEnvironments {
		files.add(".env."+env.name+".example", p.deployedEnvExample(env, p.cfg.ModuleName()), fileMode)
	}
	return files
}

func multiEnvAddOn() addOn {
	return addOn{
		enabled: has(config.MultiEnv),
		contribute: func(project) contribution {
			return contribution{readme: func(d *readmeData) { d.MultiEnv = true }}
		},
	}
}

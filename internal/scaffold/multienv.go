package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

func (e *Env) applyMultiEnvSettings(ctx context.Context) error {
	moduleName := e.cfg.ModuleName()

	data, err := e.read(e.settingsRel())
	if err != nil {
		return fmt.Errorf("read settings.py for the multi-env split: %w", err)
	}
	content := string(data)

	base, err := e.generateBasePy(content)
	if err != nil {
		return err
	}

	settingsData := multiEnvSettingsData{
		ModuleName: moduleName,
		Templates:  e.cfg.CreateTemplates(),
		AllAuth:    e.cfg.Has(config.AllAuth),
		Mailers:    config.DjangoSupportsMailers(e.cfg.Django),
		NativeCSP:  config.DjangoSupportsNativeCSP(e.cfg.Django),
		Env:        e.productionEnvExpressions(),

		FromEmailGuard: e.fromEmailGuard(""),
		Ruff:           e.cfg.Has(config.Ruff),
	}

	files := map[string]string{
		"__init__.py":   "",
		"base.py":       base,
		"local.py":      render("local.py.tmpl", settingsData),
		"production.py": render("production.py.tmpl", settingsData),
		"test.py":       render("test.py.tmpl", settingsData),
	}

	if err := e.writeFiles(filepath.Join(moduleName, "settings"), files); err != nil {
		return err
	}

	if err := os.Remove(filepath.Join(e.dir, e.settingsRel())); err != nil {
		return fmt.Errorf("remove the original settings.py: %w", err)
	}

	if err := e.updateSettingsModule(manageFile, moduleName+".settings.local"); err != nil {
		return err
	}

	if err := e.updateSettingsModule(filepath.Join(moduleName, "wsgi.py"), moduleName+".settings.production"); err != nil {
		return err
	}

	if err := e.updateSettingsModule(filepath.Join(moduleName, "asgi.py"), moduleName+".settings.production"); err != nil {
		return err
	}

	if err := e.generateEnvExampleFiles(moduleName); err != nil {
		return err
	}

	e.Step("Settings split into settings/base.py, local.py, production.py, test.py")
	e.Step("manage.py → settings.local (development)")
	e.Step("wsgi.py/asgi.py → settings.production")
	e.Step("Created: .env.production.example, .env.staging.example")
	return nil
}

func (e *Env) generateBasePy(content string) (string, error) {
	se := &settingsEditor{content: content}
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
	if e.cfg.Has(config.Pydantic) {
		return base, nil
	}
	const single = "BASE_DIR = Path(__file__).resolve().parent.parent"
	if !strings.Contains(base, single) {
		return "", &anchorError{Edit: "BASE_DIR for settings/base.py", Anchor: single}
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

func (e *Env) updateSettingsModule(rel, newModule string) error {
	data, err := e.read(rel)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(rel), err)
	}

	content := string(data)
	moduleName := e.cfg.ModuleName()

	oldSetting := regexp.MustCompile(`(['"])` + regexp.QuoteMeta(moduleName+".settings") + `(['"])`)
	at := oldSetting.FindStringSubmatchIndex(content)
	if at == nil {
		return &anchorError{Edit: "the settings module in " + filepath.Base(rel), Anchor: "'" + moduleName + ".settings'"}
	}
	quote := content[at[2]:at[3]]
	content = content[:at[0]] + quote + newModule + quote + content[at[1]:]

	mode := fileMode
	if rel == manageFile {
		mode = execMode
	}
	return e.write(rel, []byte(content), mode)
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

func (e *Env) deployedEnvExample(env deployedEnvironment, moduleName string) string {
	database := "# SQLite needs no database variables; consider PostgreSQL for production.\n"
	if vars := e.deployedDatabase().render(); vars != "" {
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
		database + e.deployedSentrySection(env.name)
}

func (e *Env) deployedDatabase() dotenvDatabase {
	if !e.cfg.Database.UsesDocker() {
		return dotenvDatabase{}
	}
	database := e.cfg.Database.Spec()
	host, port := database.Host, strconv.Itoa(database.Port)
	return dotenvDatabase{Vars: []dotenvVar{
		{"DB_NAME", dbdocker.DatabaseName(e.cfg.Name)},
		{"DB_USER", "django_user"},
		{"DB_PASSWORD", "<set-password>"},
		{"DB_HOST", host},
		{"DB_PORT", port},
	}}
}

func (e *Env) generateEnvExampleFiles(moduleName string) error {
	files := map[string]string{}
	for _, env := range deployedEnvironments {
		files[".env."+env.name+".example"] = e.deployedEnvExample(env, moduleName)
	}
	return e.writeFiles(".", files)
}

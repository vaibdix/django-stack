package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/dbdocker"
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
	host, port := "localhost", defaultPostgresPort
	switch e.Cfg.DatabaseType {
	case config.PostgresDocker:
	case config.MySQLDocker:
		host, port = mysqlTCPHost, defaultMySQLPort
	default:
		return dotenvDatabase{}
	}
	return dotenvDatabase{Vars: []dotenvVar{
		{"DB_NAME", dbdocker.DatabaseName(e.Cfg.ProjectName)},
		{"DB_USER", "django_user"},
		{"DB_PASSWORD", "<set-password>"},
		{"DB_HOST", host},
		{"DB_PORT", port},
	}}
}

func (e *Env) generateEnvExampleFiles(projectPath, moduleName string) error {
	for _, env := range deployedEnvironments {
		name := ".env." + env.name + ".example"
		if err := os.WriteFile(filepath.Join(projectPath, name), []byte(e.deployedEnvExample(env, moduleName)), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}
	return nil
}

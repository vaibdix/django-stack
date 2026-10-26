package scaffold

import (
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

type settingsTemplateData struct {
	Module           string
	Django           string
	Series           string
	Imports          []string
	BaseDir          bool
	Apps             []string
	Middleware       []string
	Templates        bool
	DefaultAutoField bool
	EnvBlock         string
	Tail             string
}

var djangoApps = []string{
	"django.contrib.admin",
	"django.contrib.auth",
	"django.contrib.contenttypes",
	"django.contrib.sessions",
	"django.contrib.messages",
	"django.contrib.staticfiles",
}

func djangoMiddleware() []string {
	return []string{
		middlewareSecurity,
		middlewareWhiteNoise,
		"django.contrib.sessions.middleware.SessionMiddleware",
		"django.middleware.common.CommonMiddleware",
		"django.middleware.csrf.CsrfViewMiddleware",
		"django.contrib.auth.middleware.AuthenticationMiddleware",
		"django.contrib.messages.middleware.MessageMiddleware",
		"django.middleware.clickjacking.XFrameOptionsMiddleware",
	}
}

func (e *Env) split() bool { return e.cfg.Has(config.MultiEnv) }

func (e *Env) settingsImports() []string {
	cfg := e.envConfig()
	var imports []string
	if !cfg.Pydantic {
		imports = append(imports, "from pathlib import Path")
	}
	imports = append(imports, cfg.importLine())
	if cfg.Pydantic {
		imports = append(imports, "from "+cfg.ModuleName+"."+strings.TrimSuffix(confFile, ".py")+" import BASE_DIR")
	} else {
		imports = append(imports, "from django.core.exceptions import ImproperlyConfigured")
	}
	if config.DjangoSupportsNativeCSP(e.cfg.Django) {
		imports = append(imports, "from django.utils.csp import CSP")
	} else {
		imports = append(imports, "from csp.constants import NONE", "from csp.constants import SELF")
	}
	if cfg.Pydantic && !e.split() {
		imports = append(imports, "from django.core.exceptions import ImproperlyConfigured")
	}
	imports = append(imports, e.contributedSettings(func(s settingsContribution) []string { return s.imports })...)
	return orderImports(imports, e.firstPartyModules())
}

func (e *Env) settingsApps() []string {
	apps := append([]string{}, djangoApps...)
	if !config.DjangoSupportsNativeCSP(e.cfg.Django) {
		apps = append(apps, appDjangoCSP)
	}
	apps = append(apps, e.contributedSettings(func(s settingsContribution) []string { return s.apps })...)
	apps = append(apps, coreApp, accountsApp)
	if e.cfg.App != "" {
		apps = append(apps, e.cfg.App)
	}
	return append(apps, e.contributedSettings(func(s settingsContribution) []string { return s.lateApps })...)
}

func (e *Env) settingsMiddleware() []string {
	middleware := djangoMiddleware()
	if config.DjangoSupportsNativeCSP(e.cfg.Django) {
		middleware = append(middleware, middlewareCSP)
	} else {
		middleware = append(middleware, middlewareDjangoCSP)
	}
	return append(middleware, e.contributedSettings(func(s settingsContribution) []string { return s.middleware })...)
}

func (e *Env) settingsEnvBlock() string {
	block := envConfigBlock(e.envConfig())
	if e.split() {
		before, after, _ := strings.Cut(block, "ALLOWED_HOSTS = ")
		_, rest, _ := strings.Cut(after, "\n")
		block = strings.TrimSuffix(before, "\n") + rest
	}
	return block
}

func (e *Env) loginSettings() string {
	switch {
	case e.cfg.Has(config.AllAuth):
		return "LOGIN_URL = \"account_login\"\n"
	case e.cfg.CreateTemplates():
		return "LOGIN_URL = \"login\"\nLOGIN_REDIRECT_URL = \"home\"\nLOGOUT_REDIRECT_URL = \"home\"\n"
	default:
		return "LOGIN_URL = \"admin:login\"\n"
	}
}

func (e *Env) settingsTail() string {
	var b strings.Builder
	b.WriteString(passwordSettings())
	b.WriteString(e.loginSettings())
	if config.DjangoSupportsNativeCSP(e.cfg.Django) {
		b.WriteString(nativeCSPReportOnly())
	} else {
		b.WriteString(djangoCSPReportOnly())
	}
	if !e.split() {
		b.WriteString(e.singleFileEnvironmentBlock())
	}
	if !e.cfg.Has(config.Structlog) {
		b.WriteString(consoleLoggingBlock())
	}
	for _, block := range e.contributedSettings(func(s settingsContribution) []string { return s.blocks }) {
		b.WriteString(block)
	}
	b.WriteString("\nAUTH_USER_MODEL = \"accounts.User\"\n")
	for _, block := range e.contributedSettings(func(s settingsContribution) []string { return s.lateBlocks }) {
		b.WriteString(block)
	}
	return b.String()
}

func (e *Env) renderSettingsModule() string {
	content := render("settings.py.tmpl", settingsTemplateData{
		Module:           e.cfg.ModuleName(),
		Django:           e.cfg.Django,
		Series:           config.DjangoSeries(e.cfg.Django),
		Imports:          e.settingsImports(),
		BaseDir:          !e.cfg.Has(config.Pydantic),
		Apps:             e.settingsApps(),
		Middleware:       e.settingsMiddleware(),
		Templates:        e.cfg.CreateTemplates(),
		DefaultAutoField: config.DjangoSeries(e.cfg.Django) == "5.2",
		EnvBlock:         e.settingsEnvBlock(),
		Tail:             e.settingsTail(),
	})
	if !e.split() {
		return content
	}
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}
	content = strings.Replace(content, "\n\n"+envBoolFunction()+"\n", "\n\n\n"+envBoolFunction()+"\n\n", 1)
	return strings.Replace(content, "BASE_DIR = Path(__file__).resolve().parent.parent", "BASE_DIR = Path(__file__).resolve().parent.parent.parent", 1)
}

type djangoEntryData struct {
	Module   string
	Series   string
	Settings string
}

func (e *Env) projectFiles() fileSet {
	module := e.cfg.ModuleName()
	local, production := module+".settings", module+".settings"
	if e.split() {
		local, production = module+".settings.local", module+".settings.production"
	}
	series := config.DjangoSeries(e.cfg.Django)

	files := fileSet{}
	files.add(manageFile, render("manage.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: local}), execMode)
	files.add(filepath.Join(module, "__init__.py"), "", fileMode)
	files.add(filepath.Join(module, "asgi.py"), render("asgi.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: production}), fileMode)
	files.add(filepath.Join(module, "wsgi.py"), render("wsgi.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: production}), fileMode)
	return files
}

func (e *Env) settingsFiles() fileSet {
	files := fileSet{}
	module := e.cfg.ModuleName()
	if !e.split() {
		files.add(filepath.Join(module, settingsFile), e.renderSettingsModule(), fileMode)
		return files
	}
	data := e.multiEnvSettingsData()
	files.addAll(filepath.Join(module, "settings"), map[string]string{
		"__init__.py":   "",
		"base.py":       e.renderSettingsModule(),
		"local.py":      render("local.py.tmpl", data),
		"production.py": render("production.py.tmpl", data),
		"test.py":       render("test.py.tmpl", data),
	})
	return files
}

func passwordSettings() string { return renderStatic("passwords.py.tmpl") }

func nativeCSPReportOnly() string { return renderStatic("csp_native.py.tmpl") }

func djangoCSPReportOnly() string { return renderStatic("csp_django.py.tmpl") }

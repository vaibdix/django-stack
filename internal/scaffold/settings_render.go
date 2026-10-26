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

func (p project) split() bool { return p.cfg.Has(config.MultiEnv) }

func (p project) settingsImports() []string {
	cfg := p.envConfig()
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
	if config.DjangoSupportsNativeCSP(p.cfg.Django) {
		imports = append(imports, "from django.utils.csp import CSP")
	} else {
		imports = append(imports, "from csp.constants import NONE", "from csp.constants import SELF")
	}
	if cfg.Pydantic && !p.split() {
		imports = append(imports, "from django.core.exceptions import ImproperlyConfigured")
	}
	imports = append(imports, p.contributedSettings(func(s settingsContribution) []string { return s.imports })...)
	return orderImports(imports, p.firstPartyModules())
}

func (p project) settingsApps() []string {
	apps := append([]string{}, djangoApps...)
	if !config.DjangoSupportsNativeCSP(p.cfg.Django) {
		apps = append(apps, appDjangoCSP)
	}
	apps = append(apps, p.contributedSettings(func(s settingsContribution) []string { return s.apps })...)
	apps = append(apps, coreApp, accountsApp)
	if p.cfg.App != "" {
		apps = append(apps, p.cfg.App)
	}
	return append(apps, p.contributedSettings(func(s settingsContribution) []string { return s.lateApps })...)
}

func (p project) settingsMiddleware() []string {
	middleware := djangoMiddleware()
	if config.DjangoSupportsNativeCSP(p.cfg.Django) {
		middleware = append(middleware, middlewareCSP)
	} else {
		middleware = append(middleware, middlewareDjangoCSP)
	}
	return append(middleware, p.contributedSettings(func(s settingsContribution) []string { return s.middleware })...)
}

func (p project) settingsEnvBlock() string {
	block := envConfigBlock(p.envConfig())
	if p.split() {
		before, after, _ := strings.Cut(block, "ALLOWED_HOSTS = ")
		_, rest, _ := strings.Cut(after, "\n")
		block = strings.TrimSuffix(before, "\n") + rest
	}
	return block
}

func (p project) loginSettings() string {
	switch {
	case p.cfg.Has(config.AllAuth):
		return "LOGIN_URL = \"account_login\"\n"
	case p.cfg.CreateTemplates():
		return "LOGIN_URL = \"login\"\nLOGIN_REDIRECT_URL = \"home\"\nLOGOUT_REDIRECT_URL = \"home\"\n"
	default:
		return "LOGIN_URL = \"admin:login\"\n"
	}
}

func (p project) settingsTail() string {
	var b strings.Builder
	b.WriteString(passwordSettings())
	b.WriteString(p.loginSettings())
	if config.DjangoSupportsNativeCSP(p.cfg.Django) {
		b.WriteString(nativeCSPReportOnly())
	} else {
		b.WriteString(djangoCSPReportOnly())
	}
	if !p.split() {
		b.WriteString(p.singleFileEnvironmentBlock())
	}
	if !p.cfg.Has(config.Structlog) {
		b.WriteString(consoleLoggingBlock())
	}
	for _, block := range p.contributedSettings(func(s settingsContribution) []string { return s.blocks }) {
		b.WriteString(block)
	}
	b.WriteString("\nAUTH_USER_MODEL = \"accounts.User\"\n")
	for _, block := range p.contributedSettings(func(s settingsContribution) []string { return s.lateBlocks }) {
		b.WriteString(block)
	}
	return b.String()
}

func (p project) renderSettingsModule() string {
	content := render("settings.py.tmpl", settingsTemplateData{
		Module:           p.cfg.ModuleName(),
		Django:           p.cfg.Django,
		Series:           config.DjangoSeries(p.cfg.Django),
		Imports:          p.settingsImports(),
		BaseDir:          !p.cfg.Has(config.Pydantic),
		Apps:             p.settingsApps(),
		Middleware:       p.settingsMiddleware(),
		Templates:        p.cfg.CreateTemplates(),
		DefaultAutoField: config.DjangoSeries(p.cfg.Django) == "5.2",
		EnvBlock:         p.settingsEnvBlock(),
		Tail:             p.settingsTail(),
	})
	if !p.split() {
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

func (p project) projectFiles() fileSet {
	module := p.cfg.ModuleName()
	local, production := module+".settings", module+".settings"
	if p.split() {
		local, production = module+".settings.local", module+".settings.production"
	}
	series := config.DjangoSeries(p.cfg.Django)

	files := fileSet{}
	files.add(manageFile, render("manage.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: local}), execMode)
	files.add(filepath.Join(module, "__init__.py"), "", fileMode)
	files.add(filepath.Join(module, "asgi.py"), render("asgi.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: production}), fileMode)
	files.add(filepath.Join(module, "wsgi.py"), render("wsgi.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: production}), fileMode)
	return files
}

func (p project) settingsFiles() fileSet {
	files := fileSet{}
	module := p.cfg.ModuleName()
	if !p.split() {
		files.add(filepath.Join(module, settingsFile), p.renderSettingsModule(), fileMode)
		return files
	}
	data := p.multiEnvSettingsData()
	files.addAll(filepath.Join(module, "settings"), map[string]string{
		"__init__.py":   "",
		"base.py":       p.renderSettingsModule(),
		"local.py":      render("local.py.tmpl", data),
		"production.py": render("production.py.tmpl", data),
		"test.py":       render("test.py.tmpl", data),
	})
	return files
}

func passwordSettings() string { return renderStatic("passwords.py.tmpl") }

func nativeCSPReportOnly() string { return renderStatic("csp_native.py.tmpl") }

func djangoCSPReportOnly() string { return renderStatic("csp_django.py.tmpl") }

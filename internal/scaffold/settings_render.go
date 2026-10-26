package scaffold

import (
	"path/filepath"
	"slices"
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
	SettingsPackage  bool
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
	if p.cfg.Django().SupportsNativeCSP() {
		imports = append(imports, "from django.utils.csp import CSP")
	} else {
		imports = append(imports, "from csp.constants import NONE", "from csp.constants import SELF")
	}
	if cfg.Pydantic && !p.split() {
		imports = append(imports, "from django.core.exceptions import ImproperlyConfigured")
	}
	imports = append(imports, contributedTo(p, func(s settingsContribution) []string { return s.imports })...)
	return orderImports(imports, p.firstPartyModules())
}

func (p project) settingsApps() ([]string, error) {
	apps := append(slices.Clone(djangoApps), coreApp, accountsApp)
	if p.cfg.App().Ok() {
		apps = append(apps, p.cfg.App().String())
	}
	return placeEntries("app", apps, contributedTo(p, func(s settingsContribution) []entry { return s.apps }))
}

func (p project) settingsMiddleware() ([]string, error) {
	entries := contributedTo(p, func(s settingsContribution) []entry { return s.middleware })
	if p.cfg.Django().SupportsNativeCSP() {
		entries = append(entries, entry{name: middlewareCSP, after: lastDjangoMiddleware, rank: rankCSP})
	}
	return placeEntries("middleware", djangoMiddleware(), entries)
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

type sectionWriter struct {
	strings.Builder
	oneBlankLine bool
}

func (w *sectionWriter) add(section string) {
	for w.oneBlankLine && strings.HasPrefix(section, "\n\n") {
		section = section[1:]
	}
	_, _ = w.WriteString(section)
}

func (p project) settingsBlocks() []block {
	blocks := []block{
		{rank: rankPasswords, text: passwordSettings()},
		{rank: rankLogin, text: p.loginSettings()},
		{rank: rankCSPReport, text: djangoCSPReportOnly()},
	}
	if p.cfg.Django().SupportsNativeCSP() {
		blocks[2].text = nativeCSPReportOnly()
	}
	if !p.split() {
		blocks = append(blocks, block{rank: rankEnvironment, text: p.singleFileEnvironmentBlock()})
	}
	if !p.cfg.Has(config.Structlog) {
		blocks = append(blocks, block{rank: rankConsoleLogging, text: consoleLoggingBlock()})
	}
	blocks = append(blocks, block{section: afterUserModel, text: "\nAUTH_USER_MODEL = \"accounts.User\"\n"})
	return append(blocks, contributedTo(p, func(s settingsContribution) []block { return s.blocks })...)
}

func (p project) settingsTail() (string, error) {
	blocks, err := orderBlocks("settings", p.settingsBlocks())
	if err != nil {
		return "", err
	}
	b := sectionWriter{oneBlankLine: p.split()}
	for _, block := range blocks {
		b.add(block.text)
	}
	return b.String(), nil
}

func (p project) renderSettingsModule() (string, error) {
	apps, err := p.settingsApps()
	if err != nil {
		return "", err
	}
	middleware, err := p.settingsMiddleware()
	if err != nil {
		return "", err
	}
	tail, err := p.settingsTail()
	if err != nil {
		return "", err
	}
	return render("settings.py.tmpl", settingsTemplateData{
		Module:           p.cfg.ModuleName(),
		Django:           p.cfg.Django().String(),
		Series:           p.cfg.Django().Series(),
		Imports:          p.settingsImports(),
		BaseDir:          !p.cfg.Has(config.Pydantic),
		Apps:             apps,
		Middleware:       middleware,
		Templates:        p.cfg.CreateTemplates(),
		DefaultAutoField: p.cfg.Django().Series() == "5.2",
		EnvBlock:         envConfigBlock(p.envConfig()),
		Tail:             tail,
		SettingsPackage:  p.split(),
	}), nil
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
	series := p.cfg.Django().Series()

	files := fileSet{}
	files.add(manageFile, render("manage.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: local}), execMode)
	files.add(filepath.Join(module, "__init__.py"), "", fileMode)
	files.add(filepath.Join(module, "asgi.py"), render("asgi.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: production}), fileMode)
	files.add(filepath.Join(module, "wsgi.py"), render("wsgi.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: production}), fileMode)
	return files
}

func (p project) settingsFiles() (fileSet, error) {
	files := fileSet{}
	module := p.cfg.ModuleName()
	settings, err := p.renderSettingsModule()
	if err != nil {
		return nil, err
	}
	if !p.split() {
		files.add(filepath.Join(module, settingsFile), settings, fileMode)
		return files, nil
	}
	data := p.multiEnvSettingsData()
	files.addAll(filepath.Join(module, "settings"), map[string]string{
		"__init__.py":   "",
		"base.py":       settings,
		"local.py":      render("local.py.tmpl", data),
		"production.py": render("production.py.tmpl", data),
		"test.py":       render("test.py.tmpl", data),
	})
	return files, nil
}

func passwordSettings() string { return renderStatic("passwords.py.tmpl") }

func nativeCSPReportOnly() string { return renderStatic("csp_native.py.tmpl") }

func djangoCSPReportOnly() string { return renderStatic("csp_django.py.tmpl") }

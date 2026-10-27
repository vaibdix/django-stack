package render

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

func (p Project) Split() bool { return p.Cfg.Has(config.MultiEnv) }

func (p Project) settingsImports() []string {
	cfg := p.envConfig()
	var imports []string
	if !cfg.Pydantic {
		imports = append(imports, "from pathlib import Path")
	}
	imports = append(imports, cfg.importLine())
	if cfg.Pydantic {
		imports = append(imports, "from "+cfg.ModuleName+"."+strings.TrimSuffix(ConfFile, ".py")+" import BASE_DIR")
	} else {
		imports = append(imports, "from django.core.exceptions import ImproperlyConfigured")
	}
	if p.Cfg.Django().SupportsNativeCSP() {
		imports = append(imports, "from django.utils.csp import CSP")
	} else {
		imports = append(imports, "from csp.constants import NONE", "from csp.constants import SELF")
	}
	if cfg.Pydantic && !p.Split() {
		imports = append(imports, "from django.core.exceptions import ImproperlyConfigured")
	}
	imports = append(imports, contributedTo(p, func(s settingsContribution) []string { return s.imports })...)
	return orderImports(imports, p.firstPartyModules())
}

func (p Project) settingsApps() ([]string, error) {
	apps := append(slices.Clone(djangoApps), coreApp, accountsApp)
	if p.Cfg.App().Ok() {
		apps = append(apps, p.Cfg.App().String())
	}
	return placeEntries("app", apps, contributedTo(p, func(s settingsContribution) []entry { return s.apps }))
}

func (p Project) settingsMiddleware() ([]string, error) {
	entries := contributedTo(p, func(s settingsContribution) []entry { return s.middleware })
	if p.Cfg.Django().SupportsNativeCSP() {
		entries = append(entries, entry{name: middlewareCSP, after: lastDjangoMiddleware, rank: rankCSP})
	}
	return placeEntries("middleware", djangoMiddleware(), entries)
}

func (p Project) loginSettings() string {
	switch {
	case p.Cfg.Has(config.AllAuth):
		return "LOGIN_URL = \"account_login\"\n"
	case p.Cfg.CreateTemplates():
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

func (p Project) settingsBlocks() []block {
	blocks := []block{
		{rank: rankPasswords, text: passwordSettings()},
		{rank: rankLogin, text: p.loginSettings()},
		{rank: rankCSPReport, text: djangoCSPReportOnly()},
	}
	if p.Cfg.Django().SupportsNativeCSP() {
		blocks[2].text = nativeCSPReportOnly()
	}
	if !p.Split() {
		blocks = append(blocks, block{rank: rankEnvironment, text: p.singleFileEnvironmentBlock()})
	}
	if !p.Cfg.Has(config.Structlog) {
		blocks = append(blocks, block{rank: rankConsoleLogging, text: consoleLoggingBlock()})
	}
	blocks = append(blocks, block{section: afterUserModel, text: "\nAUTH_USER_MODEL = \"accounts.User\"\n"})
	return append(blocks, contributedTo(p, func(s settingsContribution) []block { return s.blocks })...)
}

func (p Project) settingsTail() (string, error) {
	blocks, err := orderBlocks("settings", p.settingsBlocks())
	if err != nil {
		return "", err
	}
	b := sectionWriter{oneBlankLine: p.Split()}
	for _, block := range blocks {
		b.add(block.text)
	}
	return b.String(), nil
}

func (p Project) RenderSettingsModule() (string, error) {
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
	return Render("settings.py.tmpl", settingsTemplateData{
		Module:           p.Cfg.ModuleName(),
		Django:           p.Cfg.Django().String(),
		Series:           p.Cfg.Django().Series(),
		Imports:          p.settingsImports(),
		BaseDir:          !p.Cfg.Has(config.Pydantic),
		Apps:             apps,
		Middleware:       middleware,
		Templates:        p.Cfg.CreateTemplates(),
		DefaultAutoField: p.Cfg.Django().Series() == "5.2",
		EnvBlock:         envConfigBlock(p.envConfig()),
		Tail:             tail,
		SettingsPackage:  p.Split(),
	}), nil
}

type djangoEntryData struct {
	Module   string
	Series   string
	Settings string
}

func (p Project) projectFiles() FileSet {
	module := p.Cfg.ModuleName()
	local, production := module+".settings", module+".settings"
	if p.Split() {
		local, production = module+".settings.local", module+".settings.production"
	}
	series := p.Cfg.Django().Series()

	files := FileSet{}
	files.add(ManageFile, Render("manage.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: local}), execMode)
	files.add(filepath.Join(module, "__init__.py"), "", FileMode)
	files.add(filepath.Join(module, "asgi.py"), Render("asgi.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: production}), FileMode)
	files.add(filepath.Join(module, "wsgi.py"), Render("wsgi.py.tmpl", djangoEntryData{Module: module, Series: series, Settings: production}), FileMode)
	return files
}

func (p Project) SettingsFiles() (FileSet, error) {
	files := FileSet{}
	module := p.Cfg.ModuleName()
	settings, err := p.RenderSettingsModule()
	if err != nil {
		return nil, err
	}
	if !p.Split() {
		files.add(filepath.Join(module, SettingsFile), settings, FileMode)
		return files, nil
	}
	data := p.multiEnvSettingsData()
	files.addAll(filepath.Join(module, "settings"), map[string]string{
		"__init__.py":   "",
		"base.py":       settings,
		"local.py":      Render("local.py.tmpl", data),
		"production.py": Render("production.py.tmpl", data),
		"test.py":       Render("test.py.tmpl", data),
	})
	return files, nil
}

func passwordSettings() string { return RenderStatic("passwords.py.tmpl") }

func nativeCSPReportOnly() string { return RenderStatic("csp_native.py.tmpl") }

func djangoCSPReportOnly() string { return RenderStatic("csp_django.py.tmpl") }

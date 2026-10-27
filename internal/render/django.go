package render

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func consoleLoggingBlock() string { return RenderStatic("console_logging.py.tmpl") }

const manifestStaticStorage = "whitenoise.storage.CompressedManifestStaticFilesStorage"

const djangoDefaultFromEmail = "webmaster@localhost"

func (p Project) sendsAccountEmail() bool {
	return p.Cfg.Has(config.AllAuth) || p.Cfg.CreateTemplates()
}

func (p Project) fromEmailGuard(indent string) string {
	if !p.sendsAccountEmail() {
		return ""
	}
	return indent + `if DEFAULT_FROM_EMAIL == "` + djangoDefaultFromEmail + "\":\n" +
		indent + `    message = "DEFAULT_FROM_EMAIL is still ` + djangoDefaultFromEmail + "; set it to an address your mail server sends from.\"\n" +
		indent + "    raise ImproperlyConfigured(message)\n"
}

type environmentBlockData struct {
	Env                     ProductionEnvExpressions
	Templates               bool
	BrowserReloadApp        string
	BrowserReloadMiddleware string
	Mailers                 bool
	NativeCSP               bool
	Ruff                    bool
	PlaceholderSecretKey    string
	FromEmailGuard          string
	ManifestStaticStorage   string
}

func (p Project) singleFileEnvironmentBlock() string {
	return Render("environment.py.tmpl", environmentBlockData{
		Env:                     p.ProductionEnvExpressions(),
		Templates:               p.Cfg.CreateTemplates(),
		BrowserReloadApp:        appBrowserReload,
		BrowserReloadMiddleware: middlewareBrowserReload,
		Mailers:                 p.Cfg.Django().SupportsMailers(),
		NativeCSP:               p.Cfg.Django().SupportsNativeCSP(),
		Ruff:                    p.Cfg.Has(config.Ruff),
		PlaceholderSecretKey:    placeholderSecretKey,
		FromEmailGuard:          p.fromEmailGuard("    "),
		ManifestStaticStorage:   manifestStaticStorage,
	})
}

package scaffold

import (
	"context"
	"fmt"
	"runtime"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

func (e *Env) createDjangoProject(ctx context.Context) error {
	if err := e.emitGroup(groupProject); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Django project '%s' created.", e.cfg.Name().String()))
	return nil
}

func (e *Env) reportPerformanceTip() {
	if e.toolchain == pyenv.Pip && pyenv.IsWindows(runtime.GOOS) {
		e.Step(
			"Tip: Install 'uv' for faster package management: pip install uv")
	}
}

func consoleLoggingBlock() string { return renderStatic("console_logging.py.tmpl") }

const manifestStaticStorage = "whitenoise.storage.CompressedManifestStaticFilesStorage"

const djangoDefaultFromEmail = "webmaster@localhost"

func (p project) sendsAccountEmail() bool {
	return p.cfg.Has(config.AllAuth) || p.cfg.CreateTemplates()
}

func (p project) fromEmailGuard(indent string) string {
	if !p.sendsAccountEmail() {
		return ""
	}
	return indent + `if DEFAULT_FROM_EMAIL == "` + djangoDefaultFromEmail + "\":\n" +
		indent + `    message = "DEFAULT_FROM_EMAIL is still ` + djangoDefaultFromEmail + "; set it to an address your mail server sends from.\"\n" +
		indent + "    raise ImproperlyConfigured(message)\n"
}

type environmentBlockData struct {
	Env                     productionEnvExpressions
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

func (p project) singleFileEnvironmentBlock() string {
	return render("environment.py.tmpl", environmentBlockData{
		Env:                     p.productionEnvExpressions(),
		Templates:               p.cfg.CreateTemplates(),
		BrowserReloadApp:        appBrowserReload,
		BrowserReloadMiddleware: middlewareBrowserReload,
		Mailers:                 p.cfg.Django().SupportsMailers(),
		NativeCSP:               p.cfg.Django().SupportsNativeCSP(),
		Ruff:                    p.cfg.Has(config.Ruff),
		PlaceholderSecretKey:    placeholderSecretKey,
		FromEmailGuard:          p.fromEmailGuard("    "),
		ManifestStaticStorage:   manifestStaticStorage,
	})
}

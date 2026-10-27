package scaffold

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/render"
)

const djlintTimeout = 2 * time.Minute

func djlintCommand(t *testing.T, args ...string) (*exec.Cmd, bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), djlintTimeout)
	t.Cleanup(cancel)

	if path, err := exec.LookPath("uvx"); err == nil && !testing.Short() {
		pinned := append([]string{"--from", render.PkgDjLint + "==" + djLintVersion(), "djlint"}, args...)
		return exec.CommandContext(ctx, path, pinned...), true
	}
	if path, err := exec.LookPath("djlint"); err == nil && localDjLintIsPinned(ctx, path) {
		return exec.CommandContext(ctx, path, args...), true
	}
	return nil, false
}

func localDjLintIsPinned(ctx context.Context, path string) bool {
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	return err == nil && strings.Contains(string(out), djLintVersion())
}

func everyGeneratedTemplate(cfg render.AppTemplateConfig) map[string]string {
	opts := assets.PageOptions{
		AppName:  cfg.AppName,
		Tailwind: cfg.Tailwind,
		HTMX:     cfg.HTMX,
		API:      true,
		AllAuth:  true,
	}

	files := map[string]string{
		"templates/base.html":           assets.RenderPage(assets.PageBase, opts),
		"templates/index.html":          assets.RenderPage(assets.PageIndex, opts),
		"templates/account/login.html":  assets.AuthFormTemplate(assets.AuthPages[0], cfg.Tailwind),
		"templates/account/signup.html": assets.AuthFormTemplate(assets.AuthPages[1], cfg.Tailwind),
		"templates/account/logout.html": assets.AllAuthLogout(cfg.Tailwind),
		"app/book_list.html":            cfg.Render("book_list"),
		"app/book_form.html":            cfg.Render("book_form"),
		"app/book_delete.html":          cfg.Render("book_delete"),
	}
	files["templates/registration/login.html"] = assets.DjangoLoginTemplate(cfg.Tailwind)
	for _, page := range assets.ErrorPages {
		files["templates/"+page.Code+".html"] = assets.ErrorTemplate(page, cfg.Tailwind)
	}
	if cfg.HTMX {
		files["app/partials/book_list_items.html"] = cfg.Render("book_list_items")
		files["app/partials/book_form_partial.html"] = cfg.Render("book_form_partial")
	}
	return files
}

func djLintVersion() string { return config.Tools().DjLint }

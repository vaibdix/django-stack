package scaffold

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-cli/internal/assets"
)

const djlintTimeout = 2 * time.Minute

func djlintCommand(t *testing.T, args ...string) (*exec.Cmd, bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), djlintTimeout)
	t.Cleanup(cancel)

	if path, err := exec.LookPath("uvx"); err == nil {
		pinned := append([]string{"--from", PkgDjLint + "==" + DjLintVersion, "djlint"}, args...)
		return exec.CommandContext(ctx, path, pinned...), true
	}
	if path, err := exec.LookPath("djlint"); err == nil && localDjLintIsPinned(ctx, path) {
		return exec.CommandContext(ctx, path, args...), true
	}
	return nil, false
}

func localDjLintIsPinned(ctx context.Context, path string) bool {
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	return err == nil && strings.Contains(string(out), DjLintVersion)
}

func TestGeneratedTemplatesSurviveDjlint(t *testing.T) {
	t.Parallel()

	cmd, ok := djlintCommand(t, "--version")
	if !ok {
		t.Skip("neither djlint nor uvx is installed")
	}
	if err := cmd.Run(); err != nil {
		t.Skipf("djlint is not runnable: %v", err)
	}

	for variant, cfg := range appTemplateVariants("catalog") {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			for name, content := range everyGeneratedTemplate(cfg) {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("creating %s: %v", filepath.Dir(path), err)
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("writing %s: %v", path, err)
				}
			}

			lint, _ := djlintCommand(t, "--lint", "--profile=django", ".")
			lint.Dir = root
			out, err := lint.CombinedOutput()
			if err != nil {
				t.Errorf("djlint rejects the templates this scaffold generates, so the "+
					"pre-commit hook it also generates fails at commit zero:\n%s", out)
			}
		})
	}
}

func everyGeneratedTemplate(cfg AppTemplateConfig) map[string]string {
	opts := assets.PageOptions{
		AppName:         cfg.AppName,
		Tailwind:        cfg.Tailwind,
		HTMX:            cfg.HTMX,
		API:             true,
		AllAuth:         true,
		GlobalTemplates: true,
	}

	files := map[string]string{
		"templates/base.html":           assets.RenderPage(assets.PageBase, opts),
		"templates/index.html":          assets.RenderPage(assets.PageIndex, opts),
		"templates/account/login.html":  assets.AuthFormTemplate(assets.AuthPages[0], cfg.Tailwind),
		"templates/account/signup.html": assets.AuthFormTemplate(assets.AuthPages[1], cfg.Tailwind),
		"templates/account/logout.html": assets.AllAuthLogout(cfg.Tailwind),
		"app/book_list.html":            generateBookListTemplate(cfg),
		"app/book_form.html":            generateBookFormTemplate(cfg),
		"app/book_delete.html":          generateBookDeleteTemplate(cfg),
	}
	for _, page := range assets.ErrorPages {
		files["templates/"+page.Code+".html"] = assets.ErrorTemplate(page, cfg.Tailwind)
	}
	if cfg.HTMX {
		files["app/partials/book_list_items.html"] = generateBookListItemsPartial(cfg)
		files["app/partials/book_form_partial.html"] = generateBookFormPartial(cfg)
	}
	return files
}

func TestGeneratedTemplatesCarryNoInlineStyles(t *testing.T) {
	t.Parallel()

	for variant, cfg := range appTemplateVariants("catalog") {
		for name, content := range everyGeneratedTemplate(cfg) {
			if strings.Contains(content, `style="`) {
				t.Errorf("%s/%s carries an inline style, which the generated CSP forbids:\n%s",
					variant, name, content)
			}
		}
	}
}

func TestGeneratedTemplatesLinkByRouteName(t *testing.T) {
	t.Parallel()

	for variant, cfg := range appTemplateVariants("catalog") {
		for name, content := range everyGeneratedTemplate(cfg) {
			// 400.html and 500.html use no tags at all: a broken URLconf can be
			// the very error they report, so they cannot reverse a route.
			if name == "templates/400.html" || name == "templates/500.html" {
				continue
			}
			if strings.Contains(content, `href="/`) {
				t.Errorf("%s/%s hardcodes an internal path instead of using {%% url %%}:\n%s",
					variant, name, content)
			}
		}
	}
}

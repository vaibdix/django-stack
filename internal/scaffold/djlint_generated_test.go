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

	if path, err := exec.LookPath("djlint"); err == nil {
		return exec.CommandContext(ctx, path, args...), true
	}
	if path, err := exec.LookPath("uvx"); err == nil {
		return exec.CommandContext(ctx, path, append([]string{"djlint"}, args...)...), true
	}
	return nil, false
}

// The scaffold installs djlint and pins it in the generated pre-commit config,
// then shipped templates that fail it: 14 errors — inline styles that also
// violate the project's own Content-Security-Policy, hardcoded internal links,
// and a missing meta description. The first `git commit` in a generated project
// failed its own hook, which teaches every developer to reach for --no-verify.
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

// everyGeneratedTemplate renders every HTML file a project of this shape gets.
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
		"templates/account/login.html":  assets.AllAuthLogin(cfg.Tailwind),
		"templates/account/signup.html": assets.AllAuthSignup(cfg.Tailwind),
		"templates/account/logout.html": assets.AllAuthLogout(cfg.Tailwind),
		"app/book_list.html":            generateBookListTemplate(cfg),
		"app/book_form.html":            generateBookFormTemplate(cfg),
		"app/book_delete.html":          generateBookDeleteTemplate(cfg),
	}
	if cfg.HTMX {
		files["app/partials/book_list_items.html"] = generateBookListItemsPartial(cfg)
		files["app/partials/book_form_partial.html"] = generateBookFormPartial(cfg)
	}
	return files
}

// The Content-Security-Policy this scaffold generates sets style-src 'self', so
// an inline style attribute is not merely a lint opinion — it is markup the
// project's own policy forbids.
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

// A moved URL should be a template error, not a silent 404.
func TestGeneratedTemplatesLinkByRouteName(t *testing.T) {
	t.Parallel()

	for variant, cfg := range appTemplateVariants("catalog") {
		for name, content := range everyGeneratedTemplate(cfg) {
			if strings.Contains(content, `href="/`) {
				t.Errorf("%s/%s hardcodes an internal path instead of using {%% url %%}:\n%s",
					variant, name, content)
			}
		}
	}
}

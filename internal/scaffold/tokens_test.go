package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/assets"
	"github.com/vaibdix/django-cli/internal/config"
)

var tokenClassPattern = regexp.MustCompile(`\b(?:text|bg|border|from|via|to|ring|fill|stroke)-([a-z][a-z-]*)\b`)

var declaredTokenPattern = regexp.MustCompile(`--color-([a-z][a-z-]*)\s*:`)

func declaredTokens(t *testing.T) []string {
	t.Helper()

	css := renderStatic("tailwind-source.css.tmpl")
	theme := css
	if start := strings.Index(css, "@theme"); start >= 0 {
		theme = css[start:]
		if end := strings.Index(theme, "}"); end >= 0 {
			theme = theme[:end]
		}
	}

	var names []string
	for _, m := range declaredTokenPattern.FindAllStringSubmatch(theme, -1) {
		names = append(names, m[1])
	}
	slices.Sort(names)
	return names
}

func tailwindMarkup() map[string]string {
	cfg := AppTemplateConfig{AppName: "catalog", Tailwind: true, HTMX: true}
	opts := assets.PageOptions{AppName: "catalog", Tailwind: true, API: true, HTMX: true, AllAuth: true}

	out := map[string]string{
		"base.html":                  assets.RenderPage(assets.PageBase, opts),
		"index.html":                 assets.RenderPage(assets.PageIndex, opts),
		"api-docs.html":              assets.RenderPage(assets.PageAPIDocs, opts),
		"account/login.html":         assets.AuthFormTemplate(assets.AuthPages[0], true),
		"account/signup.html":        assets.AuthFormTemplate(assets.AuthPages[1], true),
		"account/logout.html":        assets.AllAuthLogout(true),
		"book_list.html":             generateBookListTemplate(cfg),
		"book_form.html":             generateBookFormTemplate(cfg),
		"book_delete.html":           generateBookDeleteTemplate(cfg),
		"partials/book_list_items":   generateBookListItemsPartial(cfg),
		"partials/book_form_partial": generateBookFormPartial(cfg),
		"forms.py (INPUT_CLASS)":     generateBookForms(true, true),
	}
	return out
}

func TestEveryTailwindTokenIsDeclared(t *testing.T) {
	t.Parallel()

	declared := declaredTokens(t)
	if len(declared) == 0 {
		t.Fatal("no --color-* tokens found in the generated @theme block; the whole palette would be missing")
	}

	for name, markup := range tailwindMarkup() {
		for _, m := range tokenClassPattern.FindAllStringSubmatch(markup, -1) {
			suffix := m[1]

			if !slices.Contains(declared, suffix) && isOurTokenName(suffix, declared) {
				t.Errorf("%s uses %q but --color-%s is not declared in @theme; "+
					"Tailwind emits no rule for it and the element renders unstyled",
					name, m[0], suffix)
			}
		}
	}
}

func isOurTokenName(suffix string, declared []string) bool {
	for _, name := range declared {
		if suffix == name {
			return true
		}

		if abs(len(suffix)-len(name)) <= 1 && commonPrefixLen(suffix, name) >= len(name)-1 && len(name) > 3 {
			return true
		}
	}
	return false
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func commonPrefixLen(a, b string) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func TestNoWideTrackedLabels(t *testing.T) {
	t.Parallel()

	sources := tailwindMarkup()
	cfg := AppTemplateConfig{AppName: "catalog", HTMX: true}
	sources["plain book_list.html"] = generateBookListTemplate(cfg)
	sources["plain book_form.html"] = generateBookFormTemplate(cfg)
	sources["plain book_delete.html"] = generateBookDeleteTemplate(cfg)
	sources["plain base.html"] = assets.RenderPage(assets.PageBase, assets.PageOptions{GlobalTemplates: true})
	sources["plain index.html"] = assets.RenderPage(assets.PageIndex, assets.PageOptions{GlobalTemplates: true})
	sources["base.css"] = assets.BaseCSS()
	sources["home.css"] = assets.HomeCSS()
	sources["tailwind-source.css"] = renderStatic("tailwind-source.css.tmpl")

	banned := []string{
		"uppercase",
		"text-transform: uppercase",
		"tracking-[0.2",
		"tracking-[0.3",
		"tracking-widest",
		"letter-spacing: 0.2",
		"letter-spacing: 0.3",
	}

	for name, src := range sources {
		for _, phrase := range banned {
			if strings.Contains(src, phrase) {
				t.Errorf("%s uses %q; the reference never stretches or uppercases a label", name, phrase)
			}
		}
	}
}

func TestRadiiStayCloseToTheReference(t *testing.T) {
	t.Parallel()

	sources := tailwindMarkup()
	cfg := AppTemplateConfig{AppName: "catalog", HTMX: true}
	sources["plain book_list.html"] = generateBookListTemplate(cfg)
	sources["plain book_form.html"] = generateBookFormTemplate(cfg)
	sources["plain book_delete.html"] = generateBookDeleteTemplate(cfg)
	sources["base.css"] = assets.BaseCSS()
	sources["home.css"] = assets.HomeCSS()

	oversized := []string{
		"rounded-full", "rounded-2xl", "rounded-3xl", "rounded-xl",
		"border-radius: 9999px", "border-radius: 16px", "border-radius: 28px",
		"rounded-[28px]", "rounded-[32px]",
	}

	for name, src := range sources {
		for _, phrase := range oversized {
			if strings.Contains(src, phrase) {
				t.Errorf("%s uses %q; the reference's radii are 4-8px", name, phrase)
			}
		}
	}
}

func TestGeneratedCSSUsesTheReferenceFontStack(t *testing.T) {
	t.Parallel()

	for name, css := range map[string]string{
		"base.css":            assets.BaseCSS(),
		"tailwind-source.css": renderStatic("tailwind-source.css.tmpl"),
	} {
		if !strings.Contains(css, "system-ui") {
			t.Errorf("%s does not set the system-ui stack the reference uses", name)
		}
		for _, webfont := range []string{"Inter", "@font-face", "fonts.googleapis"} {
			if strings.Contains(css, webfont) {
				t.Errorf("%s names %q but the project loads no webfont, so it silently falls back", name, webfont)
			}
		}
	}
}

func TestNoOpacityLiteralsSurviveInGeneratedMarkup(t *testing.T) {
	t.Parallel()

	for name, markup := range tailwindMarkup() {
		for _, line := range strings.Split(markup, "\n") {
			if strings.Contains(line, "white/") || strings.Contains(line, "black/") {
				t.Errorf("%s styles with an opacity literal instead of a token:\n  %s", name, strings.TrimSpace(line))
			}
		}
	}
}

func TestPlainCSSVariablesAreDeclared(t *testing.T) {
	t.Parallel()

	base := assets.BaseCSS()
	root := base
	if start := strings.Index(base, ":root"); start >= 0 {
		root = base[start:]
		if end := strings.Index(root, "}"); end >= 0 {
			root = root[:end]
		}
	}

	declared := regexp.MustCompile(`--([a-z][a-z-]*)\s*:`).FindAllStringSubmatch(root, -1)
	names := make([]string, 0, len(declared))
	for _, m := range declared {
		names = append(names, m[1])
	}
	if len(names) == 0 {
		t.Fatal("base.css declares no custom properties; every var() reference would resolve to nothing")
	}

	used := regexp.MustCompile(`var\(--([a-z][a-z-]*)\)`)
	sources := map[string]string{
		"base.css": base,
		"home.css": assets.HomeCSS(),
	}
	cfg := AppTemplateConfig{AppName: "catalog", HTMX: true}
	sources["book_list.html"] = generateBookListTemplate(cfg)
	sources["book_form.html"] = generateBookFormTemplate(cfg)
	sources["book_delete.html"] = generateBookDeleteTemplate(cfg)
	sources["account/login.html"] = assets.AuthFormTemplate(assets.AuthPages[0], false)
	sources["account/signup.html"] = assets.AuthFormTemplate(assets.AuthPages[1], false)
	sources["account/logout.html"] = assets.AllAuthLogout(false)

	for name, src := range sources {
		for _, m := range used.FindAllStringSubmatch(src, -1) {
			if !slices.Contains(names, m[1]) {
				t.Errorf("%s references var(--%s), which base.css does not declare; it resolves to nothing",
					name, m[1])
			}
		}
	}
}

func TestStylesheetsDoNotCarryRulesForRemovedMarkup(t *testing.T) {
	t.Parallel()

	rendered := strings.Join([]string{
		assets.RenderPage(assets.PageBase, assets.PageOptions{GlobalTemplates: true}),
		assets.RenderPage(assets.PageIndex, assets.PageOptions{GlobalTemplates: true}),
		assets.RenderPage(assets.PageAPIDocs, assets.PageOptions{GlobalTemplates: true}),
	}, "\n")

	base := stripCSSComments(assets.BaseCSS())
	home := stripCSSComments(assets.HomeCSS())

	for _, class := range []string{
		"footer-content", "footer-brand", "footer-description", "footer-links",
		"hero-grid", "features-subtitle", "feature-icon",
	} {
		inCSS := strings.Contains(base, "."+class) || strings.Contains(home, "."+class)
		inHTML := strings.Contains(rendered, class)
		if inCSS && !inHTML {
			t.Errorf(".%s is styled but never rendered; it is a rule for markup that no longer exists", class)
		}
	}
}

var cssCommentPattern = regexp.MustCompile(`(?s)/\*.*?\*/`)

func stripCSSComments(css string) string {
	return cssCommentPattern.ReplaceAllString(css, "")
}

func TestPlainTemplatesCarryNoTailwindUtilities(t *testing.T) {
	t.Parallel()

	cfg := AppTemplateConfig{AppName: "catalog", HTMX: true}
	pages := map[string]string{
		"book_list.html":   generateBookListTemplate(cfg),
		"book_form.html":   generateBookFormTemplate(cfg),
		"book_delete.html": generateBookDeleteTemplate(cfg),
	}

	for name, page := range pages {
		for _, utility := range []string{"bg-surface", "text-heading", "border-rule", "text-body"} {
			if strings.Contains(page, utility) {
				t.Errorf("plain %s uses the Tailwind utility %q, which no stylesheet defines "+
					"in a project generated without Tailwind", name, utility)
			}
		}
	}
}

func TestGeneratedTemplatesAreDiscoverableForFormatting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	appDir := filepath.Join(root, "catalog")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("creating app dir: %v", err)
	}

	e := &Env{Cfg: config.Project{
		ProjectName: "my-shop", AppName: "catalog", CreateTemplates: true,
		SetupRestFramework: true, SetupTailwind: true, SetupHTMX: true,
	}}
	if err := e.createExampleAPI(root, generateAPIPackage(e.Cfg.AppName, e.Cfg.SetupPydanticConf)); err != nil {
		t.Fatalf("createExampleAPI: %v", err)
	}

	files, err := findTemplateFilesForFormatting(root)
	if err != nil {
		t.Fatalf("findTemplateFilesForFormatting: %v", err)
	}

	var onDisk int
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".html" {
			onDisk++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking generated project: %v", err)
	}

	if onDisk == 0 {
		t.Fatal("the example app generated no templates at all")
	}
	if len(files) != onDisk {
		t.Errorf("the formatting pass found %d templates but the project contains %d; "+
			"the difference ships unformatted", len(files), onDisk)
	}
}

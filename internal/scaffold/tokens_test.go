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

// The design tokens are the one part of the reskin that fails silently.
//
// Tailwind generates a utility for each --color-<name> declared in @theme. Use
// `text-body` without declaring `--color-body` and Tailwind emits no rule at
// all: the class lands in the HTML, matches nothing, and the element renders
// with inherited colours. No build error, no runtime error — just an element
// that looks subtly wrong on a page nobody has opened yet.
//
// So the two have to be checked against each other, and the opacity literals
// the tokens replaced have to stay gone.

// tokenClassPattern matches the colour utilities that resolve through @theme.
// Tailwind's own scales (text-sm, border-b) are excluded by requiring the
// suffix to be one of our token names, which is what tokenNames provides.
var tokenClassPattern = regexp.MustCompile(`\b(?:text|bg|border|from|via|to|ring|fill|stroke)-([a-z][a-z-]*)\b`)

// declaredTokenPattern finds the --color-<name> declarations in the @theme
// block the scaffolder writes into a generated project.
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

// tailwindMarkup returns every piece of Tailwind-styled markup the scaffolder
// can emit, keyed by a name that identifies it in a failure.
func tailwindMarkup() map[string]string {
	cfg := AppTemplateConfig{AppName: "catalog", Tailwind: true, HTMX: true}
	opts := assets.PageOptions{AppName: "catalog", Tailwind: true, API: true, HTMX: true, AllAuth: true}

	out := map[string]string{
		"base.html":                  assets.RenderPage(assets.PageBase, opts),
		"index.html":                 assets.RenderPage(assets.PageIndex, opts),
		"api-docs.html":              assets.RenderPage(assets.PageAPIDocs, opts),
		"account/login.html":         assets.AllAuthLogin(true),
		"account/signup.html":        assets.AllAuthSignup(true),
		"account/logout.html":        assets.AllAuthLogout(true),
		"book_list.html":             generateBookListTemplate(cfg),
		"book_form.html":             generateBookFormTemplate(cfg),
		"book_delete.html":           generateBookDeleteTemplate(cfg),
		"partials/book_list_items":   generateBookListItemsPartial(cfg),
		"partials/book_form_partial": generateBookFormPartial(cfg),
		"forms.py (INPUT_CLASS)":     generateBookForms(true),
	}
	return out
}

// TestEveryTailwindTokenIsDeclared is the check that closes the silent gap: a
// custom colour utility used in markup must have a matching @theme declaration.
func TestEveryTailwindTokenIsDeclared(t *testing.T) {
	t.Parallel()

	declared := declaredTokens(t)
	if len(declared) == 0 {
		t.Fatal("no --color-* tokens found in the generated @theme block; the whole palette would be missing")
	}

	for name, markup := range tailwindMarkup() {
		for _, m := range tokenClassPattern.FindAllStringSubmatch(markup, -1) {
			suffix := m[1]
			// Only the names that look like ours are checked; Tailwind's own
			// scales and keywords are not this test's business.
			if !slices.Contains(declared, suffix) && isOurTokenName(suffix, declared) {
				t.Errorf("%s uses %q but --color-%s is not declared in @theme; "+
					"Tailwind emits no rule for it and the element renders unstyled",
					name, m[0], suffix)
			}
		}
	}
}

// isOurTokenName reports whether suffix looks like one of this project's token
// names rather than a Tailwind built-in. A name is ours if it is a token, or if
// it is a near-miss of one — which is exactly the typo case worth catching.
func isOurTokenName(suffix string, declared []string) bool {
	for _, name := range declared {
		if suffix == name {
			return true
		}
		// A single-character difference in either direction: `text-headin`,
		// `bg-surfaces`. Anything further away is a Tailwind class.
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

// The tokens exist so that colour is named rather than described. A stray
// opacity literal is not wrong in itself, but it is invisible to the palette:
// changing --color-rule will not move it.
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

// The plain-CSS variant has the same requirement expressed differently: it
// names CSS custom properties, and those must be declared in base.css.
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
	sources["account/login.html"] = assets.AllAuthLogin(false)
	sources["account/signup.html"] = assets.AllAuthSignup(false)
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

// The stylesheets carry rules for markup that content.go emits. A rule for
// markup nobody renders is dead weight that the next reader has to check.
func TestStylesheetsDoNotCarryRulesForRemovedMarkup(t *testing.T) {
	t.Parallel()

	rendered := strings.Join([]string{
		assets.RenderPage(assets.PageBase, assets.PageOptions{GlobalTemplates: true}),
		assets.RenderPage(assets.PageIndex, assets.PageOptions{GlobalTemplates: true}),
		assets.RenderPage(assets.PageAPIDocs, assets.PageOptions{GlobalTemplates: true}),
	}, "\n")

	// Comments are stripped first. A stylesheet's own note that it dropped a
	// rule mentions that rule by name, and a checker that cannot tell prose
	// from a selector reports the explanation as the problem.
	base := stripCSSComments(assets.BaseCSS())
	home := stripCSSComments(assets.HomeCSS())

	// The classes the previous design left behind after its markup changed.
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

// The scaffolder must not offer a Tailwind-only class to a project that has no
// Tailwind build. The plain templates are selected by AppTemplateConfig.style,
// so this guards that selection rather than the markup itself.
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

// findTemplateFilesForFormatting drives the mandatory djlint pass. It must see
// every generated template, or the reskin ships unformatted markup.
func TestGeneratedTemplatesAreDiscoverableForFormatting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	appDir := filepath.Join(root, "catalog")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("creating app dir: %v", err)
	}

	e := &Env{Cfg: config.Project{
		ProjectName: "my-shop", AppName: "catalog",
		SetupRestFramework: true, SetupTailwind: true, SetupHTMX: true,
	}}
	if err := e.createExampleAPI(root); err != nil {
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

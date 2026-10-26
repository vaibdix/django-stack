package scaffold

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/report"
)

var tailwindSourceLine = regexp.MustCompile(`(?m)^@source "([^"]+)";$`)

func globPattern(glob string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); i++ {
		switch {
		case strings.HasPrefix(glob[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case glob[i] == '*':
			b.WriteString("[^/]*")
		default:
			b.WriteString(regexp.QuoteMeta(string(glob[i])))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

func TestTailwindScansEveryFileThatCarriesClasses(t *testing.T) {
	t.Parallel()

	source := renderStatic("tailwind-source.css.tmpl")
	matches := tailwindSourceLine.FindAllStringSubmatch(source, -1)
	if len(matches) == 0 {
		t.Fatalf("the Tailwind source declares no @source:\n%s", source)
	}

	patterns := make([]*regexp.Regexp, 0, len(matches))
	for _, m := range matches {
		resolved := path.Clean(path.Join(tailwindSourceDir, m[1]))
		patterns = append(patterns, globPattern(resolved))
	}

	for _, file := range []string{
		"templates/base.html",
		"templates/account/login.html",
		"catalog/templates/catalog/book_list.html",
		"catalog/templates/catalog/partials/book_list_items.html",
		"catalog/forms.py",
	} {
		covered := false
		for _, pattern := range patterns {
			covered = covered || pattern.MatchString(file)
		}
		if !covered {
			t.Errorf("%s is not matched by any @source %v, so its classes are never generated", file, matches)
		}
	}
}

func TestTailwindInputIsNotAStaticFile(t *testing.T) {
	t.Parallel()

	if tailwindSourceDir == "static" || strings.HasPrefix(tailwindSourceDir, "static/") {
		t.Fatalf("the Tailwind input sits in %s, inside STATICFILES_DIRS: collectstatic copies it and "+
			"the manifest storage aborts on its @import \"tailwindcss\"", tailwindSourceDir)
	}

	projectPath := t.TempDir()
	e := &Env{cfg: mustSpec(projectForStack("tailwind", "")), reporter: report.Nop{}}
	if err := e.at(projectPath).ensurePackageJSON(); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}
	if err := e.at(projectPath).updatePackageJSONForTailwind(); err != nil {
		t.Fatalf("updatePackageJSONForTailwind: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(projectPath, "package.json"))
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	want := "-i ./" + tailwindSourceDir + "/" + tailwindSourceFile + " -o ./static/dist/styles.css"
	if !strings.Contains(string(manifest), want) {
		t.Errorf("package.json does not build %q:\n%s", want, manifest)
	}
}

func TestThemeBlocksAreTopLevel(t *testing.T) {
	t.Parallel()

	source := renderStatic("tailwind-source.css.tmpl")
	depth, inComment := 0, false
	for i := 0; i < len(source); i++ {
		switch {
		case inComment:
			inComment = !strings.HasPrefix(source[i:], "*/")
		case source[i] == '"':

			i += strings.IndexByte(source[i+1:], '"') + 1
		case strings.HasPrefix(source[i:], "/*"):
			inComment = true
		case source[i] == '{':
			depth++
		case source[i] == '}':
			depth--
		case strings.HasPrefix(source[i:], "@theme") && depth > 0:
			t.Fatalf("@theme is nested at depth %d; Tailwind hoists it to the top level, where it overwrites "+
				"the light palette for every visitor. Override the custom properties on :root instead.", depth)
		}
	}
}

func TestDarkModeOverridesThePaletteOnRoot(t *testing.T) {
	t.Parallel()

	source := renderStatic("tailwind-source.css.tmpl")
	at := strings.Index(source, "@media (prefers-color-scheme: dark)")
	if at == -1 {
		t.Fatal("the Tailwind source has no dark-mode media query")
	}
	dark := source[at:]
	for _, token := range declaredTokens(t) {
		if !strings.Contains(dark, "--color-"+token+":") {
			t.Errorf("--color-%s has no dark value, so it stays light in dark mode", token)
		}
	}
}

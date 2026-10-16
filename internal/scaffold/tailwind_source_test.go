package scaffold

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
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

	var patterns []*regexp.Regexp
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
	e := &Env{Cfg: projectForStack("tailwind", ""), Reporter: NopReporter{}}
	if err := e.ensurePackageJSON(projectPath); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}
	if err := e.updatePackageJSONForTailwind(projectPath); err != nil {
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

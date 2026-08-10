package scaffold

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

// Template rendering for generated project files.
//
// Everything this package writes into a user's project — Python modules, Django
// templates, shell scripts, YAML config — lives in templates/ as a real file of
// its own type, not as a raw string literal spliced together with string
// concatenation. That buys three things the previous arrangement did not have:
// the content is syntax-highlighted and lintable as what it actually is, the
// conditional variants sit next to each other instead of being separate
// near-identical literals, and a template that fails to parse is caught at
// startup rather than written to disk.

//go:embed templates
var templateFS embed.FS

// Templates use [[ ]] rather than Go's default {{ }}.
//
// Most of what is generated here is Django template source, which is built out
// of {{ variable }} and {% tag %}. Using Go's defaults would mean escaping every
// one of them. [[ ]] appears nowhere in any generated artifact, so a single
// delimiter choice covers Python, HTML, YAML and shell alike — there is no
// per-file rule to remember and no way to pick the wrong one.
const (
	templateLeftDelim  = "[["
	templateRightDelim = "]]"
)

// templates holds every embedded template, parsed once.
//
// template.Must is appropriate here: the templates are embedded at build time
// and cannot be influenced by user input, so a parse failure is a programmer
// error that should stop the program immediately and identically on every
// machine. TestEveryTemplateParsesAndRenders exercises this in CI, so a broken
// template fails a build rather than a user's scaffold.
var templates = template.Must(
	template.New("scaffold").
		Delims(templateLeftDelim, templateRightDelim).
		ParseFS(templateFS, "templates/*/*"),
)

// render executes the named template and returns its output.
//
// It panics rather than returning an error for the same reason template.Must is
// used above: both the template and the data type are compile-time constants of
// this package, so a failure here means the binary is broken, not that the
// user did something wrong. Returning an error would force every generator —
// all of which are pure string functions — to grow an error return for a
// condition that no caller could do anything about.
func render(name string, data any) string {
	var b strings.Builder
	if err := templates.ExecuteTemplate(&b, name, data); err != nil {
		panic(fmt.Sprintf("scaffold: rendering embedded template %q: %v", name, err))
	}
	return b.String()
}

// renderStatic returns a template that takes no data.
func renderStatic(name string) string { return render(name, nil) }

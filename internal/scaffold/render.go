package scaffold

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates
var templateFS embed.FS

const (
	templateLeftDelim  = "[["
	templateRightDelim = "]]"
)

var templates = template.Must(
	template.New("scaffold").
		Delims(templateLeftDelim, templateRightDelim).
		ParseFS(templateFS, "templates/*/*"),
)

func render(name string, data any) string {
	var b strings.Builder
	if err := templates.ExecuteTemplate(&b, name, data); err != nil {
		panic(fmt.Sprintf("scaffold: rendering embedded template %q: %v", name, err))
	}
	return b.String()
}

func renderStatic(name string) string { return render(name, nil) }

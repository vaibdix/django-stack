package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"
	"text/template"
)

//go:embed templates
var templateFS embed.FS

const (
	LeftDelim  = "[["
	RightDelim = "]]"
)

var templates = template.Must(
	template.New("assets").
		Delims(LeftDelim, RightDelim).
		ParseFS(templateFS, "templates/*/*"),
)

var rendered sync.Map

func Render(name string, data any) string {
	rendered.Store(name, struct{}{})
	var b strings.Builder
	if err := templates.ExecuteTemplate(&b, name, data); err != nil {
		panic(fmt.Sprintf("assets: rendering embedded template %q: %v", name, err))
	}
	return b.String()
}

func TemplateNames() []string {
	var names []string
	_ = fs.WalkDir(templateFS, "templates", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			names = append(names, path.Base(p))
		}
		return err
	})
	slices.Sort(names)
	return names
}

func TemplateFS() fs.FS { return templateFS }

func Unrendered() []string {
	var missing []string
	for _, name := range TemplateNames() {
		if strings.TrimSpace(templates.Lookup(name).Root.String()) == "" {
			continue
		}
		if _, ok := rendered.Load(name); !ok {
			missing = append(missing, name)
		}
	}
	return missing
}
